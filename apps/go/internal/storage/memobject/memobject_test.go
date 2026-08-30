package memobject_test

import (
	"context"
	"testing"

	"github.com/0muji4/Runa/apps/go/internal/storage"
	"github.com/0muji4/Runa/apps/go/internal/storage/memobject"
	"github.com/0muji4/Runa/apps/go/internal/storage/objecttest"
)

// TestMemObjectMeetsTheContract runs the ObjectStore contract suite against the
// in-memory fake; internal/storage runs the same suite against a real MinIO.
func TestMemObjectMeetsTheContract(t *testing.T) {
	t.Parallel()
	objecttest.RunObjectStoreSuite(t, func(t *testing.T) storage.ObjectStore {
		t.Helper()
		return &writableStore{memobject.New()}
	})
}

// writableStore adapts the fake's Put to the suite's seeding seam.
type writableStore struct{ *memobject.Store }

func (w *writableStore) PutForTest(_ context.Context, key string, body []byte, contentType string) error {
	w.Put(key, storage.ObjectInfo{Size: int64(len(body)), ContentType: contentType})
	return nil
}
