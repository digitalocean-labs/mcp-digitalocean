package cache

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestCache_SetAndGet(t *testing.T) {
	cache := New(1*time.Second, true)
	
	// Test setting and getting a value
	cache.Set("key1", "value1")
	
	value, found := cache.Get("key1")
	if !found {
		t.Error("Expected to find key1")
	}
	
	if value != "value1" {
		t.Errorf("Expected value1, got %v", value)
	}
}

func TestCache_Expiration(t *testing.T) {
	cache := New(100*time.Millisecond, true)
	
	cache.Set("key1", "value1")
	
	// Should be available immediately
	_, found := cache.Get("key1")
	if !found {
		t.Error("Expected to find key1 immediately")
	}
	
	// Wait for expiration
	time.Sleep(150 * time.Millisecond)
	
	_, found = cache.Get("key1")
	if found {
		t.Error("Expected key1 to be expired")
	}
}

func TestCache_Delete(t *testing.T) {
	cache := New(1*time.Second, true)
	
	cache.Set("key1", "value1")
	cache.Delete("key1")
	
	_, found := cache.Get("key1")
	if found {
		t.Error("Expected key1 to be deleted")
	}
}

func TestCache_Clear(t *testing.T) {
	cache := New(1*time.Second, true)
	
	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	
	if cache.Size() != 2 {
		t.Errorf("Expected size 2, got %d", cache.Size())
	}
	
	cache.Clear()
	
	if cache.Size() != 0 {
		t.Errorf("Expected size 0 after clear, got %d", cache.Size())
	}
}

func TestCache_Disabled(t *testing.T) {
	cache := New(1*time.Second, false)
	
	cache.Set("key1", "value1")
	
	_, found := cache.Get("key1")
	if found {
		t.Error("Expected cache to be disabled")
	}
	
	if cache.Size() != 0 {
		t.Errorf("Expected size 0 for disabled cache, got %d", cache.Size())
	}
}

func TestCache_WithCache(t *testing.T) {
	cache := New(1*time.Second, true)
	callCount := 0
	
	fn := cache.WithCache("test_key", func(ctx context.Context, args ...interface{}) (interface{}, error) {
		callCount++
		return "result", nil
	})
	
	// First call should execute the function
	result1, err := fn(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if result1 != "result" {
		t.Errorf("Expected 'result', got %v", result1)
	}
	
	if callCount != 1 {
		t.Errorf("Expected function to be called once, got %d", callCount)
	}
	
	// Second call should use cache
	result2, err := fn(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if result2 != "result" {
		t.Errorf("Expected 'result', got %v", result2)
	}
	
	if callCount != 1 {
		t.Errorf("Expected function to still be called once (cached), got %d", callCount)
	}
}

func TestCache_ConcurrentAccess(t *testing.T) {
	cache := New(1*time.Second, true)
	
	// Test concurrent writes and reads
	done := make(chan bool, 10)
	
	// Start multiple goroutines writing to cache
	for i := 0; i < 5; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				cache.Set(fmt.Sprintf("key_%d_%d", id, j), fmt.Sprintf("value_%d_%d", id, j))
			}
			done <- true
		}(i)
	}
	
	// Start multiple goroutines reading from cache
	for i := 0; i < 5; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				cache.Get(fmt.Sprintf("key_%d_%d", id, j))
			}
			done <- true
		}(i)
	}
	
	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
	
	// Cache should still be functional
	cache.Set("test", "value")
	value, found := cache.Get("test")
	if !found || value != "value" {
		t.Error("Cache should still be functional after concurrent access")
	}
}
