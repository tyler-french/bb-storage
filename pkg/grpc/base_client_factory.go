package grpc

import (
	"context"
	"crypto/tls"
	"math"
	"net/http"
	"net/url"

	configuration "github.com/buildbarn/bb-storage/pkg/proto/configuration/grpc"
	"github.com/buildbarn/bb-storage/pkg/util"
	"github.com/jmespath/go-jmespath"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/oauth"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/security/advancedtls"
	"google.golang.org/grpc/status"
)

type baseClientFactory struct {
	dialer             ClientDialer
	dialOptions        []grpc.DialOption
	unaryInterceptors  []grpc.UnaryClientInterceptor
	streamInterceptors []grpc.StreamClientInterceptor
}

// NewBaseClientFactory creates factory for gRPC clients that calls into
// ClientDialer to construct the actual client.
func NewBaseClientFactory(
	dialer ClientDialer,
	dialOptions []grpc.DialOption,
	unaryInterceptors []grpc.UnaryClientInterceptor,
	streamInterceptors []grpc.StreamClientInterceptor,
) ClientFactory {
	// Limit slice capacity to length, so that any appending
	// additional interceptors always triggers a copy.
	return baseClientFactory{
		dialer:             dialer,
		dialOptions:        dialOptions[:len(dialOptions):len(dialOptions)],
		unaryInterceptors:  unaryInterceptors[:len(unaryInterceptors):len(unaryInterceptors)],
		streamInterceptors: streamInterceptors[:len(streamInterceptors):len(streamInterceptors)],
	}
}

func newClientDialOptionsFromTLSConfig(tlsConfig *tls.Config) ([]grpc.DialOption, error) {
	if tlsConfig == nil {
		return []grpc.DialOption{grpc.WithInsecure()}, nil
	}

	opts := advancedtls.Options{
		MinTLSVersion: tlsConfig.MinVersion,
		MaxTLSVersion: tlsConfig.MaxVersion,
		CipherSuites:  tlsConfig.CipherSuites,
		IdentityOptions: advancedtls.IdentityCertificateOptions{
			GetIdentityCertificatesForClient: tlsConfig.GetClientCertificate,
		},
		RootOptions: advancedtls.RootCertificateOptions{
			RootCertificates: tlsConfig.RootCAs,
		},
	}
	// advancedtls checks MinTLSVersion > MaxTLSVersion before applying
	// defaults:
	// https://github.com/grpc/grpc-go/blob/master/security/advancedtls/advancedtls.go#L243-L245
	// If setting a default minimum, set math.MaxUint16 as the Max to get around
	// this check.
	if opts.MaxTLSVersion == 0 && opts.MinTLSVersion >= 0 {
		opts.MaxTLSVersion = math.MaxUint16
	}

	tc, err := advancedtls.NewClientCreds(&opts)
	if err != nil {
		return nil, util.StatusWrapWithCode(err, codes.InvalidArgument, "Failed to configure GRPC client TLS")
	}
	dialOptions := []grpc.DialOption{grpc.WithTransportCredentials(tc)}
	if tlsConfig.ServerName != "" {
		dialOptions = append(dialOptions, grpc.WithAuthority(tlsConfig.ServerName))
	}

	return dialOptions, nil
}

func (cf baseClientFactory) NewClientFromConfiguration(config *configuration.ClientConfiguration) (grpc.ClientConnInterface, error) {
	if config == nil {
		return nil, status.Error(codes.InvalidArgument, "No gRPC client configuration provided")
	}

	dialOptions := cf.dialOptions
	unaryInterceptors := cf.unaryInterceptors
	streamInterceptors := cf.streamInterceptors

	// Optional: Tracing attributes.
	if tracing := config.Tracing; len(tracing) > 0 {
		extractor := NewProtoTraceAttributesExtractor(tracing, util.DefaultErrorLogger)
		unaryInterceptors = append(unaryInterceptors, extractor.InterceptUnaryClient)
		streamInterceptors = append(streamInterceptors, extractor.InterceptStreamClient)
	}

	// Optional: TLS.
	tlsConfig, err := util.NewTLSConfigFromClientConfiguration(config.Tls)
	if err != nil {
		return nil, util.StatusWrap(err, "Failed to create TLS configuration")
	}
	tlsDialOpts, err := newClientDialOptionsFromTLSConfig(tlsConfig)
	if err != nil {
		return nil, util.StatusWrap(err, "Failed to convert TLS configuration")
	}
	dialOptions = append(dialOptions, tlsDialOpts...)

	if windowSize := config.InitialWindowSizeBytes; windowSize != 0 {
		dialOptions = append(dialOptions, grpc.WithInitialWindowSize(windowSize))
	}
	if connWindowSize := config.InitialConnWindowSizeBytes; connWindowSize != 0 {
		dialOptions = append(dialOptions, grpc.WithInitialConnWindowSize(connWindowSize))
	}

	// Optional: OAuth authentication.
	if oauthConfig := config.Oauth; oauthConfig != nil {
		var perRPC credentials.PerRPCCredentials
		var err error
		switch credentials := oauthConfig.Credentials.(type) {
		case *configuration.ClientOAuthConfiguration_GoogleDefaultCredentials:
			perRPC, err = oauth.NewApplicationDefault(context.Background(), oauthConfig.Scopes...)
		case *configuration.ClientOAuthConfiguration_ServiceAccountKey:
			perRPC, err = oauth.NewServiceAccountFromKey([]byte(credentials.ServiceAccountKey), oauthConfig.Scopes...)
		default:
			return nil, status.Error(codes.InvalidArgument, "gRPC client credentials are wrong: one of googleDefaultCredentials or serviceAccountKey should be provided")
		}
		if err != nil {
			return nil, util.StatusWrap(err, "Failed to create gRPC credentials")
		}
		dialOptions = append(dialOptions, grpc.WithPerRPCCredentials(perRPC))
	}

	// Optional: Keepalive.
	if config.Keepalive != nil {
		time := config.Keepalive.Time
		if err := time.CheckValid(); err != nil {
			return nil, util.StatusWrap(err, "Failed to parse keepalive time")
		}
		timeout := config.Keepalive.Timeout
		if err := timeout.CheckValid(); err != nil {
			return nil, util.StatusWrap(err, "Failed to parse keepalive timeout")
		}
		dialOptions = append(dialOptions, grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                time.AsDuration(),
			Timeout:             timeout.AsDuration(),
			PermitWithoutStream: config.Keepalive.PermitWithoutStream,
		}))
	}

	// Optional: add metadata.
	if headers := config.AddMetadata; len(headers) > 0 {
		var headerValues MetadataHeaderValues
		for _, entry := range headers {
			headerValues.Add(entry.Header, entry.Values)
		}
		unaryInterceptors = append(
			unaryInterceptors,
			NewMetadataAddingUnaryClientInterceptor(headerValues))
		streamInterceptors = append(
			streamInterceptors,
			NewMetadataAddingStreamClientInterceptor(headerValues))
	}

	// Optional: proxying.
	if proxyURL := config.ProxyUrl; proxyURL != "" {
		parsedProxyURL, err := url.Parse(proxyURL)
		if err != nil {
			return nil, util.StatusWrap(err, "Failed to parse proxy URL")
		}
		proxyDialer := proxyDialer{
			httpProxy: http.ProxyURL(parsedProxyURL),
		}
		dialOptions = append(dialOptions, grpc.WithContextDialer(proxyDialer.proxyDial))
	}

	// Optional: metadata extraction.
	if jmesExpression := config.AddMetadataJmespathExpression; jmesExpression != "" {
		expr, err := jmespath.Compile(jmesExpression)
		if err != nil {
			return nil, util.StatusWrap(err, "Failed to compile JMESPath expression")
		}
		extractor, err := NewJMESPathMetadataExtractor(expr)
		if err != nil {
			return nil, util.StatusWrap(err, "Failed to create JMESPath extractor")
		}
		unaryInterceptors = append(
			unaryInterceptors,
			NewMetadataExtractingAndForwardingUnaryClientInterceptor(extractor))
		streamInterceptors = append(
			streamInterceptors,
			NewMetadataExtractingAndForwardingStreamClientInterceptor(extractor))
	}

	// Optional: service config.
	if serviceConfig := config.DefaultServiceConfig; serviceConfig != nil {
		serviceConfigJSON, err := serviceConfig.MarshalJSON()
		if err != nil {
			return nil, util.StatusWrap(err, "Failed to marshal default service config")
		}
		dialOptions = append(
			dialOptions,
			grpc.WithDefaultServiceConfig(string(serviceConfigJSON)))
	}

	dialOptions = append(
		dialOptions,
		grpc.WithChainUnaryInterceptor(unaryInterceptors...),
		grpc.WithChainStreamInterceptor(streamInterceptors...))
	return cf.dialer(context.Background(), config.Address, dialOptions...)
}
