package informer

import (
	"context"
	"sync"

	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

type informer struct {
	cache *cache
}

// NewInformer returns a new instance of an informer object.
func NewInformer() *informer {
	return &informer{
		cache: &cache{
			locker: sync.Mutex{},
		},
	}
}

// Create is the create function to satisfy the handlers.EventHandler interface.
func (informer *informer) Create(ctx context.Context, create event.CreateEvent, queue workqueue.RateLimitingInterface) {
	informer.cache.Add(create.Object)
}

// Update is the update function to satisfy the handlers.EventHandler interface.
func (informer *informer) Update(ctx context.Context, update event.UpdateEvent, queue workqueue.RateLimitingInterface) {
	informer.cache.Add(update.ObjectNew)
}

// Delete is the delete function to satisfy the handlers.EventHandler interface.
func (informer *informer) Delete(ctx context.Context, delete event.DeleteEvent, queue workqueue.RateLimitingInterface) {
	informer.cache.Remove(delete.Object)
}

// Generic is the generic function to satisfy the handlers.EventHandler interface.
func (informer *informer) Generic(ctx context.Context, generic event.GenericEvent, queue workqueue.RateLimitingInterface) {
	informer.cache.Add(generic.Object)
}
