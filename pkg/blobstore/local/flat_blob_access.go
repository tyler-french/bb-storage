package local

import (
	"context"
	"sync"
	"time"

	"github.com/buildbarn/bb-storage/pkg/blobstore"
	"github.com/buildbarn/bb-storage/pkg/blobstore/buffer"
	"github.com/buildbarn/bb-storage/pkg/blobstore/slicing"
	"github.com/buildbarn/bb-storage/pkg/capabilities"
	"github.com/buildbarn/bb-storage/pkg/digest"
	"github.com/buildbarn/bb-storage/pkg/util"
	"github.com/prometheus/client_golang/prometheus"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	flatBlobAccessPrometheusMetrics sync.Once

	flatBlobAccessRefreshesBlobs = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "buildbarn",
			Subsystem: "blobstore",
			Name:      "flat_blob_access_refreshes_blobs",
			Help:      "The number of blobs that were refreshed when requested",
			Buckets:   append([]float64{0}, prometheus.ExponentialBuckets(1.0, 2.0, 16)...),
		},
		[]string{"storage_type", "operation"})

	flatBlobAccessRefreshesDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "buildbarn",
			Subsystem: "blobstore",
			Name:      "flat_blob_access_refreshes_duration_seconds",
			Help:      "Time spent refreshing blobs in seconds",
			Buckets:   util.DecimalExponentialBuckets(-3, 6, 2),
		},
		[]string{"storage_type", "operation"})

	flatBlobAccessRefreshesSizeBytes = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "buildbarn",
			Subsystem: "blobstore",
			Name:      "flat_blob_access_refreshes_size_bytes",
			Help:      "Size of blobs being refreshed in bytes",
			Buckets:   prometheus.ExponentialBuckets(1.0, 2.0, 33),
		},
		[]string{"storage_type", "operation"})
)

type flatBlobAccess struct {
	capabilities.Provider

	keyLocationMap  KeyLocationMap
	locationBlobMap LocationBlobMap
	digestKeyFormat digest.KeyFormat

	lock        *ShardedLock
	refreshLock sync.Mutex

	refreshesBlobsGet              prometheus.Observer
	refreshesBlobsGetFromComposite prometheus.Observer
	refreshesBlobsFindMissing      prometheus.Observer

	refreshesBlobsDurationGet              prometheus.Observer
	refreshesBlobsDurationGetFromComposite prometheus.Observer
	refreshesBlobsDurationFindMissing      prometheus.Observer
	refreshesBlobsSizeGet                  prometheus.Observer
	refreshesBlbosSizeGetFromComposite     prometheus.Observer
	refreshesBlobsSizeFindMissing          prometheus.Observer
}

// NewFlatBlobAccess creates a BlobAccess that forwards all calls to
// KeyLocationMap and LocationBlobMap backend. It's called 'flat',
// because it assumes all objects are stored in a flat namespace. It
// either ignores the REv2 instance name in digests entirely, or it
// strongly partitions objects by instance name. It does not introduce
// any hierarchy.
func NewFlatBlobAccess(keyLocationMap KeyLocationMap, locationBlobMap LocationBlobMap, digestKeyFormat digest.KeyFormat, lock *sync.RWMutex, storageType string, capabilitiesProvider capabilities.Provider) blobstore.BlobAccess {
	flatBlobAccessPrometheusMetrics.Do(func() {
		prometheus.MustRegister(flatBlobAccessRefreshesBlobs)
		prometheus.MustRegister(flatBlobAccessRefreshesDurationSeconds)
		prometheus.MustRegister(flatBlobAccessRefreshesSizeBytes)
	})

	// Create a sharded lock with 64 shards for optimal concurrency
	// We ignore the provided lock parameter to use a sharded lock instead
	shardedLock := NewShardedLock(64)

	return &flatBlobAccess{
		Provider: capabilitiesProvider,

		keyLocationMap:  keyLocationMap,
		locationBlobMap: locationBlobMap,
		digestKeyFormat: digestKeyFormat,
		lock:            shardedLock,

		refreshesBlobsGet:              flatBlobAccessRefreshesBlobs.WithLabelValues(storageType, "Get"),
		refreshesBlobsGetFromComposite: flatBlobAccessRefreshesBlobs.WithLabelValues(storageType, "GetFromComposite"),
		refreshesBlobsFindMissing:      flatBlobAccessRefreshesBlobs.WithLabelValues(storageType, "FindMissing"),

		refreshesBlobsDurationGet:              flatBlobAccessRefreshesDurationSeconds.WithLabelValues(storageType, "Get"),
		refreshesBlobsDurationGetFromComposite: flatBlobAccessRefreshesDurationSeconds.WithLabelValues(storageType, "GetFromComposite"),
		refreshesBlobsDurationFindMissing:      flatBlobAccessRefreshesDurationSeconds.WithLabelValues(storageType, "FindMissing"),
		refreshesBlobsSizeGet:                  flatBlobAccessRefreshesSizeBytes.WithLabelValues(storageType, "Get"),
		refreshesBlbosSizeGetFromComposite:     flatBlobAccessRefreshesSizeBytes.WithLabelValues(storageType, "GetFromComposite"),
		refreshesBlobsSizeFindMissing:          flatBlobAccessRefreshesSizeBytes.WithLabelValues(storageType, "FindMissing"),
	}
}

func (ba *flatBlobAccess) getKey(digest digest.Digest) Key {
	return NewKeyFromString(digest.GetKey(ba.digestKeyFormat))
}

// finalizePut is called to finalize a write to the data store. This
// method must be called while holding the write lock.
func (ba *flatBlobAccess) finalizePut(putFinalizer LocationBlobPutFinalizer, key Key) (Location, error) {
	location, err := putFinalizer()
	if err != nil {
		return Location{}, err
	}
	return location, ba.keyLocationMap.Put(key, location)
}

func (ba *flatBlobAccess) Get(ctx context.Context, blobDigest digest.Digest) buffer.Buffer {
	key := ba.getKey(blobDigest)

	// Look up the blob in storage while holding a read lock.
	ba.lock.RLock(key)
	location, err := ba.keyLocationMap.Get(key)
	if err != nil {
		ba.lock.RUnlock(key)
		return buffer.NewBufferFromError(err)
	}
	getter, needsRefresh := ba.locationBlobMap.Get(location)
	if !needsRefresh {
		// The blob doesn't need to be refreshed, so we can
		// return its data directly.
		b := getter(blobDigest)
		ba.lock.RUnlock(key)
		return b
	}
	ba.lock.RUnlock(key)

	// Blob was found, but it needs to be refreshed to ensure it
	// doesn't disappear. Retry loading the blob a second time, this
	// time holding a write lock. This allows us to mutate the
	// key-location map or allocate new space to copy the blob on
	// the fly.
	//
	// TODO: Instead of copying data on the fly, should this be done
	// immediately, so that we can prevent potential duplication by
	// picking up the refresh lock?
	refreshStart := time.Now()

	ba.lock.Lock(key)
	location, err = ba.keyLocationMap.Get(key)
	if err != nil {
		ba.lock.Unlock(key)
		return buffer.NewBufferFromError(err)
	}
	getter, needsRefresh = ba.locationBlobMap.Get(location)
	b := getter(blobDigest)
	if !needsRefresh {
		// Some other thread managed to refresh the blob before
		// we got the write lock. No need to copy anymore.
		ba.lock.Unlock(key)
		return b
	}

	// Allocate space for the copy.
	putWriter, err := ba.locationBlobMap.Put(location.SizeBytes)
	ba.lock.Unlock(key)
	if err != nil {
		b.Discard()
		return buffer.NewBufferFromError(util.StatusWrap(err, "Failed to refresh blob"))
	}

	// Copy the object while it's been returned. Block until copying
	// has finished to apply back-pressure.
	b1, b2 := b.CloneStream()
	return b1.WithTask(func() error {
		putFinalizer := putWriter(b2)
		ba.lock.Lock(key)
		_, err := ba.finalizePut(putFinalizer, key)
		if err == nil {
			ba.refreshesBlobsGet.Observe(1)
			ba.refreshesBlobsSizeGet.Observe(float64(location.SizeBytes))
			ba.refreshesBlobsDurationGet.Observe(time.Since(refreshStart).Seconds())
		}
		ba.lock.Unlock(key)
		if err != nil {
			return util.StatusWrap(err, "Failed to refresh blob")
		}
		return nil
	})
}

func (ba *flatBlobAccess) GetFromComposite(ctx context.Context, parentDigest, childDigest digest.Digest, slicer slicing.BlobSlicer) buffer.Buffer {
	parentKey := ba.getKey(parentDigest)
	childKey := ba.getKey(childDigest)

	// Look up the blob in storage while holding a read lock. Even
	// though the child object determines the data to be returned,
	// the parent object controls whether it needs to be refreshed.
	// We therefore look up both unconditionally.
	ba.lock.RLock(parentKey)
	parentLocation, err := ba.keyLocationMap.Get(parentKey)
	if err != nil {
		ba.lock.RUnlock(parentKey)
		return buffer.NewBufferFromError(err)
	}
	if _, needsRefresh := ba.locationBlobMap.Get(parentLocation); !needsRefresh {
		ba.lock.RLock(childKey)
		if childLocation, err := ba.keyLocationMap.Get(childKey); err == nil {
			// The parent object doesn't need to be
			// refreshed, and the child object exists.
			// Return the child object immediately.
			childGetter, _ := ba.locationBlobMap.Get(childLocation)
			b := childGetter(childDigest)
			ba.lock.RUnlock(childKey)
			ba.lock.RUnlock(parentKey)
			return b
		} else if status.Code(err) != codes.NotFound {
			ba.lock.RUnlock(childKey)
			ba.lock.RUnlock(parentKey)
			return buffer.NewBufferFromError(err)
		}
		ba.lock.RUnlock(childKey)
	}
	ba.lock.RUnlock(parentKey)

	// The parent object was found, but it either hasn't been sliced
	// yet, or it needs to be refreshed to ensure it doesn't
	// disappear. Retry the process above, but now with write locks
	// acquired.
	ba.refreshLock.Lock()
	defer ba.refreshLock.Unlock()

	ba.lock.Lock(parentKey)
	parentLocation, err = ba.keyLocationMap.Get(parentKey)
	if err != nil {
		ba.lock.Unlock(parentKey)
		return buffer.NewBufferFromError(err)
	}

	var bParentSlicing buffer.Buffer
	var putFinalizer LocationBlobPutFinalizer
	parentGetter, needsRefresh := ba.locationBlobMap.Get(parentLocation)
	// Add refresh start time
	refreshStart := time.Now()
	if needsRefresh {
		// The parent object needs to be refreshed and sliced.
		bParent := parentGetter(parentDigest)
		putWriter, err := ba.locationBlobMap.Put(parentLocation.SizeBytes)
		ba.lock.Unlock(parentKey)
		if err != nil {
			bParent.Discard()
			return buffer.NewBufferFromError(util.StatusWrap(err, "Failed to refresh blob"))
		}

		// Copy the data while it's being sliced.
		bParent1, bParent2 := bParent.CloneStream()
		bParentSlicing = bParent1.WithTask(func() error {
			putFinalizer = putWriter(bParent2)
			return nil
		})
	} else {
		ba.lock.RLock(childKey)
		if childLocation, err := ba.keyLocationMap.Get(childKey); err == nil {
			// The parent object was refreshed and sliced in
			// the meantime.
			childGetter, _ := ba.locationBlobMap.Get(childLocation)
			b := childGetter(childDigest)
			ba.lock.RUnlock(childKey)
			ba.lock.Unlock(parentKey)
			return b
		} else if status.Code(err) != codes.NotFound {
			ba.lock.RUnlock(childKey)
			ba.lock.Unlock(parentKey)
			return buffer.NewBufferFromError(err)
		}
		ba.lock.RUnlock(childKey)

		// The parent object only needs to be sliced.
		bParentSlicing = parentGetter(parentDigest)
		ba.lock.Unlock(parentKey)
	}

	// Perform the slicing.
	bChild, slices := slicer.Slice(bParentSlicing, childDigest)
	sliceKeys := make([]Key, 0, len(slices))
	for _, slice := range slices {
		sliceKeys = append(sliceKeys, ba.getKey(slice.Digest))
	}

	// Complete refreshing in case it was performed.
	ba.lock.Lock(parentKey)
	if needsRefresh {
		parentLocation, err = ba.finalizePut(putFinalizer, parentKey)
		// Add size metric before refresh
		ba.refreshesBlbosSizeGetFromComposite.Observe(float64(parentLocation.SizeBytes))
		if err != nil {
			ba.lock.Unlock(parentKey)
			bChild.Discard()
			return buffer.NewBufferFromError(util.StatusWrap(err, "Failed to refresh blob"))
		}
		ba.refreshesBlobsDurationGetFromComposite.Observe(time.Since(refreshStart).Seconds())
		ba.refreshesBlobsGetFromComposite.Observe(1)
	}

	// Create key-location map entries for each of the slices. This
	// permits subsequent GetFromComposite() calls to access the
	// individual parts without any slicing.
	for i, slice := range slices {
		sliceKey := sliceKeys[i]
		ba.lock.Lock(sliceKey)
		err := ba.keyLocationMap.Put(sliceKey, Location{
			BlockIndex:  parentLocation.BlockIndex,
			OffsetBytes: parentLocation.OffsetBytes + slice.OffsetBytes,
			SizeBytes:   slice.SizeBytes,
		})
		ba.lock.Unlock(sliceKey)
		if err != nil {
			ba.lock.Unlock(parentKey)
			bChild.Discard()
			return buffer.NewBufferFromError(util.StatusWrapf(err, "Failed to create child blob %#v", slice.Digest.String()))
		}
	}
	ba.lock.Unlock(parentKey)
	return bChild
}

func (ba *flatBlobAccess) Put(ctx context.Context, blobDigest digest.Digest, b buffer.Buffer) error {
	sizeBytes, err := b.GetSizeBytes()
	if err != nil {
		b.Discard()
		return err
	}

	// We don't need to hold a lock for a specific key when allocating space,
	// as this operation deals with global resources
	putWriter, err := ba.locationBlobMap.Put(sizeBytes)
	if err != nil {
		b.Discard()
		return err
	}

	// Ingest the data associated with the object. This must be done
	// without holding any locks, so that I/O can happen in
	// parallel.
	putFinalizer := putWriter(b)

	key := ba.getKey(blobDigest)
	ba.lock.Lock(key)
	_, err = ba.finalizePut(putFinalizer, key)
	ba.lock.Unlock(key)
	return err
}

func (ba *flatBlobAccess) FindMissing(ctx context.Context, digests digest.Set) (digest.Set, error) {
	// Convert all digests to Keys.
	keys := make([]Key, 0, digests.Length())
	for _, blobDigest := range digests.Items() {
		keys = append(keys, ba.getKey(blobDigest))
	}

	// Perform an initial scan to determine which blobs are present
	// in storage.
	type blobToRefresh struct {
		digest digest.Digest
		key    Key
	}
	var blobsToRefresh []blobToRefresh
	missing := digest.NewSetBuilder()

	// Check each key individually with its own lock
	for i, blobDigest := range digests.Items() {
		key := keys[i]
		ba.lock.RLock(key)
		if location, err := ba.keyLocationMap.Get(key); err == nil {
			_, needsRefresh := ba.locationBlobMap.Get(location)
			if needsRefresh {
				// Blob is present, but it must be
				// refreshed for it to remain present.
				// Enqueue it for the second scan.
				blobsToRefresh = append(blobsToRefresh, blobToRefresh{
					digest: blobDigest,
					key:    key,
				})
			}
		} else if status.Code(err) == codes.NotFound {
			// Blob is absent.
			missing.Add(blobDigest)
		} else {
			ba.lock.RUnlock(key)
			return digest.EmptySet, util.StatusWrapf(err, "Failed to get blob %#v", blobDigest.String())
		}
		ba.lock.RUnlock(key)
	}

	if len(blobsToRefresh) == 0 {
		return missing.Build(), nil
	}

	// One or more blobs need to be refreshed.
	//
	// We should prevent concurrent FindMissing() calls from
	// refreshing the same blobs, as that would cause data to be
	// duplicated and load to increase significantly. Pick up the
	// refresh lock to ensure bandwidth of refreshing is limited to
	// one thread.
	ba.refreshLock.Lock()
	defer ba.refreshLock.Unlock()

	// Add refresh start time before the refresh loop
	refreshStart := time.Now()
	blobsRefreshedSuccessfully := 0
	var blobRefreshSizeBytes int64

	for _, blobToRefresh := range blobsToRefresh {
		ba.lock.Lock(blobToRefresh.key)
		if location, err := ba.keyLocationMap.Get(blobToRefresh.key); err == nil {
			getter, needsRefresh := ba.locationBlobMap.Get(location)
			if needsRefresh {
				// Blob is present and still needs to be
				// refreshed. Allocate space for a copy.
				b := getter(blobToRefresh.digest)
				blobRefreshSizeBytes += location.SizeBytes
				putWriter, err := ba.locationBlobMap.Put(location.SizeBytes)
				ba.lock.Unlock(blobToRefresh.key)
				if err != nil {
					b.Discard()
					return digest.EmptySet, util.StatusWrapf(err, "Failed to refresh blob %#v", blobToRefresh.digest.String())
				}

				// Copy the data while unlocked, so that
				// concurrent requests for other data
				// continue to be serviced.
				putFinalizer := putWriter(b)

				ba.lock.Lock(blobToRefresh.key)
				if _, err := ba.finalizePut(putFinalizer, blobToRefresh.key); err != nil {
					ba.lock.Unlock(blobToRefresh.key)
					return digest.EmptySet, util.StatusWrapf(err, "Failed to refresh blob %#v", blobToRefresh.digest.String())
				}
				blobsRefreshedSuccessfully++
			}
		} else if status.Code(err) == codes.NotFound {
			// Blob disappeared between the first and second
			// scan. Simply report it as missing.
			missing.Add(blobToRefresh.digest)
		} else {
			ba.lock.Unlock(blobToRefresh.key)
			return digest.EmptySet, util.StatusWrapf(err, "Failed to get blob %#v", blobToRefresh.digest.String())
		}
		ba.lock.Unlock(blobToRefresh.key)
	}

	ba.refreshesBlobsFindMissing.Observe(float64(blobsRefreshedSuccessfully))
	ba.refreshesBlobsDurationFindMissing.Observe(time.Since(refreshStart).Seconds())
	ba.refreshesBlobsSizeFindMissing.Observe(float64(blobRefreshSizeBytes))
	return missing.Build(), nil
}
