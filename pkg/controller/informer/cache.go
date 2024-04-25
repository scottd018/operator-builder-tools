package informer

import (
	"fmt"
	"sync"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type cache struct {
	resources map[string]client.Object
	locker    sync.Mutex
}

// Add adds a resource to the cache.
func (c *cache) Add(resource client.Object) {
	c.locker.Lock()
	defer c.locker.Unlock()

	// ensure we do not have a nil value for our resources
	if c.resources == nil {
		c.resources = map[string]client.Object{}
	}

	c.resources[cacheKey(resource)] = resource
}

// Remove removes a resource from the cache.
func (c *cache) Remove(resource client.Object) {
	c.locker.Lock()
	defer c.locker.Unlock()

	// return immediately if we have no resources in the cache as there is nothing to remove
	if len(c.resources) == 0 {
		return
	}

	delete(c.resources, cacheKey(resource))
}

// Find finds a resource on the cache and returns a nil value if it has no resource.
func (c *cache) Find(resource client.Object) client.Object {
	c.locker.Lock()
	defer c.locker.Unlock()

	for existing := range c.resources {
		if cacheKey(resource) == cacheKey(c.resources[existing]) {
			return c.resources[existing]
		}
	}

	return nil
}

// Has determines if the cache has an existing resource.
func (c *cache) Has(resource client.Object) bool {
	return c.Find(resource) != nil
}

// cacheKey returns the cache key for a particular resource.  It is a unique key that ensures that the associated
// resource belongs to this key.
func cacheKey(resource client.Object) string {
	return fmt.Sprintf("%s/%s/%s/%s%s",
		resource.GetObjectKind().GroupVersionKind().Group,
		resource.GetObjectKind().GroupVersionKind().Version,
		resource.GetObjectKind().GroupVersionKind().Kind,
		resource.GetNamespace(),
		resource.GetName(),
	)
}
