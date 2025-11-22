package ai

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/glive/infrastructure/cache"
)

// AICache caches AI analysis responses
type AICache struct {
	cache    *cache.LRUCache
	ttl      time.Duration
	hashFunc func(input AnalysisInput) string
}

type cacheEntry struct {
	output    *AnalysisOutput
	createdAt time.Time
}

type comprehensiveCacheEntry struct {
	analysis  *ComprehensiveAnalysis
	createdAt time.Time
}

// NewAICache creates a new AI cache
func NewAICache(capacity int, ttl time.Duration) *AICache {
	return &AICache{
		cache: cache.NewLRUCache(capacity, ttl),
		ttl:    ttl,
		hashFunc: func(input AnalysisInput) string {
			return hashAnalysisInput(input)
		},
	}
}

// DefaultAICache creates an AI cache with default settings
func DefaultAICache() *AICache {
	return NewAICache(100, 24*time.Hour)
}

// Get retrieves a cached analysis output
func (c *AICache) Get(input AnalysisInput) (*AnalysisOutput, bool) {
	key := c.hashFunc(input)
	val, ok := c.cache.Get(key)
	if !ok {
		return nil, false
	}

	entry, ok := val.(*cacheEntry)
	if !ok {
		return nil, false
	}

	// Check if entry is still valid
	if time.Since(entry.createdAt) >= c.ttl {
		c.cache.Delete(key)
		return nil, false
	}

	return entry.output, true
}

// Set stores an analysis output in the cache
func (c *AICache) Set(input AnalysisInput, output *AnalysisOutput) {
	key := c.hashFunc(input)
	entry := &cacheEntry{
		output:    output,
		createdAt: time.Now(),
	}
	c.cache.Set(key, entry)
}

// Delete removes an entry from the cache
func (c *AICache) Delete(input AnalysisInput) {
	key := c.hashFunc(input)
	c.cache.Delete(key)
}

// Clear clears all entries from the cache
func (c *AICache) Clear() {
	// The LRUCache doesn't have a Clear method, so we'll need to work around it
	// For now, we'll just create a new cache
	c.cache = cache.NewLRUCache(100, c.ttl)
}

// hashAnalysisInput creates a hash key from analysis input
func hashAnalysisInput(input AnalysisInput) string {
	// Create a deterministic hash from the input
	// Include: project path, readme content, and file list
	hash := sha256.New()
	hash.Write([]byte(input.ProjectPath))
	hash.Write([]byte("\n"))
	hash.Write([]byte(input.ReadmeContent))
	hash.Write([]byte("\n"))
	
	// Hash file list (sorted for consistency)
	for _, file := range input.FileList {
		hash.Write([]byte(file))
		hash.Write([]byte("\n"))
	}

	return hex.EncodeToString(hash.Sum(nil))
}

// HashWithCommitHash creates a cache key including commit hash
func HashWithCommitHash(repoURL string, commitHash string, fileHashes map[string]string) string {
	hash := sha256.New()
	hash.Write([]byte(repoURL))
	hash.Write([]byte("\n"))
	hash.Write([]byte(commitHash))
	hash.Write([]byte("\n"))
	
	// Include file hashes for change detection
	for file, fileHash := range fileHashes {
		hash.Write([]byte(file))
		hash.Write([]byte(":"))
		hash.Write([]byte(fileHash))
		hash.Write([]byte("\n"))
	}

	return hex.EncodeToString(hash.Sum(nil))
}

// GetWithCommitHash retrieves cached analysis using commit hash
func (c *AICache) GetWithCommitHash(key string) (*AnalysisOutput, bool) {
	val, ok := c.cache.Get(key)
	if !ok {
		return nil, false
	}

	entry, ok := val.(*cacheEntry)
	if !ok {
		return nil, false
	}

	// Check if entry is still valid
	if time.Since(entry.createdAt) >= c.ttl {
		c.cache.Delete(key)
		return nil, false
	}

	return entry.output, true
}

// SetWithCommitHash stores analysis using commit hash key
func (c *AICache) SetWithCommitHash(key string, output *AnalysisOutput) {
	entry := &cacheEntry{
		output:    output,
		createdAt: time.Now(),
	}
	c.cache.Set(key, entry)
}

// GetComprehensiveAnalysis retrieves cached comprehensive analysis
func (c *AICache) GetComprehensiveAnalysis(projectPath string) (*ComprehensiveAnalysis, bool) {
	key := hashProjectPath(projectPath)
	val, ok := c.cache.Get(key)
	if !ok {
		return nil, false
	}

	entry, ok := val.(*comprehensiveCacheEntry)
	if !ok {
		return nil, false
	}

	// Check if entry is still valid
	if time.Since(entry.createdAt) >= c.ttl {
		c.cache.Delete(key)
		return nil, false
	}

	return entry.analysis, true
}

// SetComprehensiveAnalysis stores comprehensive analysis in cache
func (c *AICache) SetComprehensiveAnalysis(projectPath string, analysis *ComprehensiveAnalysis) {
	key := hashProjectPath(projectPath)
	entry := &comprehensiveCacheEntry{
		analysis:  analysis,
		createdAt: time.Now(),
	}
	c.cache.Set(key, entry)
}

// hashProjectPath creates a hash key from project path
func hashProjectPath(projectPath string) string {
	hash := sha256.New()
	hash.Write([]byte(projectPath))
	return hex.EncodeToString(hash.Sum(nil))
}

