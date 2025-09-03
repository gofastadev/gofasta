// Package core provides concurrent file I/O and project structure handling.
// This implements Phase 1.3d: Build concurrent file I/O and project structure handling.
package core

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// FileHandler provides concurrent file I/O and project structure handling
type FileHandler struct {
	config     *FileHandlerConfig
	fileCache  map[string]*CachedFile
	projects   map[string]*ProjectStructure
	operations chan FileOperation
	mu         sync.RWMutex
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc

	// Metrics
	reads      int64
	writes     int64
	deletes    int64
	cacheHits  int64
	cacheMisses int64
	bytesRead  int64
	bytesWritten int64
}

// FileHandlerConfig contains configuration for file handler
type FileHandlerConfig struct {
	// I/O settings
	MaxConcurrentOps   int
	BufferSize         int
	EnableCache        bool
	CacheTTL           time.Duration
	MaxCacheSize       int64 // in bytes
	
	// Project settings
	RootDir            string
	IgnorePatterns     []string
	FollowSymlinks     bool
	WatchForChanges    bool
	
	// Performance settings
	ParallelReads      bool
	ParallelWrites     bool
	BatchOperations    bool
	CompressionEnabled bool
	
	// Safety settings
	BackupBeforeWrite  bool
	AtomicWrites       bool
	ValidatePermissions bool
}

// CachedFile represents a cached file
type CachedFile struct {
	Path       string
	Content    []byte
	Hash       string
	Size       int64
	ModTime    time.Time
	CachedAt   time.Time
	AccessCount int64
	Compressed bool
}

// ProjectStructure represents a project's file structure
type ProjectStructure struct {
	RootPath    string
	Files       map[string]*FileMetadata
	Directories map[string]*DirectoryInfo
	Packages    map[string]*PackageInfo
	ScannedAt   time.Time
	TotalSize   int64
	FileCount   int
}

// FileMetadata contains information about a file
type FileMetadata struct {
	Path       string
	Name       string
	Size       int64
	ModTime    time.Time
	IsSymlink  bool
	LinkTarget string
	Hash       string
	MimeType   string
	Language   string // "go", "gofa", etc.
}

// DirectoryInfo contains information about a directory
type DirectoryInfo struct {
	Path      string
	Name      string
	FileCount int
	Subdirs   []string
	Size      int64
}

// PackageInfo contains information about a package
type PackageInfo struct {
	Path        string
	Name        string
	ImportPath  string
	GoFiles     []string
	GofaFiles   []string
	TestFiles   []string
	Dependencies []string
}

// FileOperation represents a file operation
type FileOperation struct {
	Type      OperationType
	Path      string
	Content   []byte
	Options   FileOptions
	Result    chan OperationResult
}

// OperationType represents the type of file operation
type OperationType int

const (
	OpRead OperationType = iota
	OpWrite
	OpDelete
	OpCopy
	OpMove
	OpMkdir
	OpList
	OpStat
)

// FileOptions contains options for file operations
type FileOptions struct {
	CreateDirs   bool
	Overwrite    bool
	Permissions  fs.FileMode
	Backup       bool
	Atomic       bool
}

// OperationResult contains the result of a file operation
type OperationResult struct {
	Success  bool
	Data     []byte
	Info     *FileMetadata
	Error    error
	Duration time.Duration
}

// DefaultFileHandlerConfig returns the default configuration
func DefaultFileHandlerConfig() *FileHandlerConfig {
	return &FileHandlerConfig{
		MaxConcurrentOps:    10,
		BufferSize:          4096,
		EnableCache:         true,
		CacheTTL:            5 * time.Minute,
		MaxCacheSize:        100 * 1024 * 1024, // 100MB
		RootDir:             ".",
		IgnorePatterns:      []string{".git", "node_modules", "vendor", "*.tmp"},
		FollowSymlinks:      false,
		WatchForChanges:     false,
		ParallelReads:       true,
		ParallelWrites:      false,
		BatchOperations:     true,
		CompressionEnabled:  false,
		BackupBeforeWrite:   false,
		AtomicWrites:        true,
		ValidatePermissions: true,
	}
}

// NewFileHandler creates a new file handler
func NewFileHandler(config *FileHandlerConfig) *FileHandler {
	if config == nil {
		config = DefaultFileHandlerConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	fh := &FileHandler{
		config:     config,
		fileCache:  make(map[string]*CachedFile),
		projects:   make(map[string]*ProjectStructure),
		operations: make(chan FileOperation, config.MaxConcurrentOps),
		ctx:        ctx,
		cancel:     cancel,
	}

	// Start worker pool
	for i := 0; i < config.MaxConcurrentOps; i++ {
		fh.wg.Add(1)
		go fh.worker()
	}

	// Start cache cleaner if enabled
	if config.EnableCache {
		go fh.cacheCleaner()
	}

	return fh
}

// worker processes file operations
func (fh *FileHandler) worker() {
	defer fh.wg.Done()

	for {
		select {
		case op := <-fh.operations:
			fh.processOperation(op)
		case <-fh.ctx.Done():
			return
		}
	}
}

// processOperation processes a single file operation
func (fh *FileHandler) processOperation(op FileOperation) {
	start := time.Now()
	var result OperationResult

	switch op.Type {
	case OpRead:
		result = fh.readOperation(op)
	case OpWrite:
		result = fh.writeOperation(op)
	case OpDelete:
		result = fh.deleteOperation(op)
	case OpCopy:
		result = fh.copyOperation(op)
	case OpMove:
		result = fh.moveOperation(op)
	case OpMkdir:
		result = fh.mkdirOperation(op)
	case OpList:
		result = fh.listOperation(op)
	case OpStat:
		result = fh.statOperation(op)
	default:
		result = OperationResult{
			Success: false,
			Error:   fmt.Errorf("unknown operation type: %v", op.Type),
		}
	}

	result.Duration = time.Since(start)
	op.Result <- result
}

// ReadFile reads a file
func (fh *FileHandler) ReadFile(path string) ([]byte, error) {
	// Check cache first
	if fh.config.EnableCache {
		if cached := fh.getFromCache(path); cached != nil {
			atomic.AddInt64(&fh.cacheHits, 1)
			return cached.Content, nil
		}
		atomic.AddInt64(&fh.cacheMisses, 1)
	}

	// Create operation
	op := FileOperation{
		Type:   OpRead,
		Path:   path,
		Result: make(chan OperationResult, 1),
	}

	// Send operation
	select {
	case fh.operations <- op:
	case <-fh.ctx.Done():
		return nil, fmt.Errorf("file handler shutting down")
	}

	// Wait for result
	result := <-op.Result
	if !result.Success {
		return nil, result.Error
	}

	return result.Data, nil
}

// WriteFile writes a file
func (fh *FileHandler) WriteFile(path string, content []byte, options FileOptions) error {
	op := FileOperation{
		Type:    OpWrite,
		Path:    path,
		Content: content,
		Options: options,
		Result:  make(chan OperationResult, 1),
	}

	select {
	case fh.operations <- op:
	case <-fh.ctx.Done():
		return fmt.Errorf("file handler shutting down")
	}

	result := <-op.Result
	if !result.Success {
		return result.Error
	}

	return nil
}

// DeleteFile deletes a file
func (fh *FileHandler) DeleteFile(path string) error {
	op := FileOperation{
		Type:   OpDelete,
		Path:   path,
		Result: make(chan OperationResult, 1),
	}

	select {
	case fh.operations <- op:
	case <-fh.ctx.Done():
		return fmt.Errorf("file handler shutting down")
	}

	result := <-op.Result
	if !result.Success {
		return result.Error
	}

	// Remove from cache
	if fh.config.EnableCache {
		fh.removeFromCache(path)
	}

	return nil
}

// CopyFile copies a file
func (fh *FileHandler) CopyFile(src, dst string, options FileOptions) error {
	op := FileOperation{
		Type:    OpCopy,
		Path:    src,
		Content: []byte(dst), // Use content field for destination
		Options: options,
		Result:  make(chan OperationResult, 1),
	}

	select {
	case fh.operations <- op:
	case <-fh.ctx.Done():
		return fmt.Errorf("file handler shutting down")
	}

	result := <-op.Result
	if !result.Success {
		return result.Error
	}

	return nil
}

// ScanProject scans a project directory structure
func (fh *FileHandler) ScanProject(rootPath string) (*ProjectStructure, error) {
	// Check if already scanned
	fh.mu.RLock()
	if project, exists := fh.projects[rootPath]; exists {
		// Check if scan is recent
		if time.Since(project.ScannedAt) < time.Minute {
			fh.mu.RUnlock()
			return project, nil
		}
	}
	fh.mu.RUnlock()

	project := &ProjectStructure{
		RootPath:    rootPath,
		Files:       make(map[string]*FileMetadata),
		Directories: make(map[string]*DirectoryInfo),
		Packages:    make(map[string]*PackageInfo),
		ScannedAt:   time.Now(),
	}

	err := filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Continue walking
		}

		// Check ignore patterns
		if fh.shouldIgnore(path) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if d.IsDir() {
			dirInfo := &DirectoryInfo{
				Path: path,
				Name: d.Name(),
			}
			project.Directories[path] = dirInfo
		} else {
			info, err := d.Info()
			if err != nil {
				return nil
			}

			fileInfo := &FileMetadata{
				Path:    path,
				Name:    d.Name(),
				Size:    info.Size(),
				ModTime: info.ModTime(),
			}

			// Detect file type
			ext := filepath.Ext(path)
			switch ext {
			case ".go":
				fileInfo.Language = "go"
			case ".gofa":
				fileInfo.Language = "gofa"
			}

			project.Files[path] = fileInfo
			project.FileCount++
			project.TotalSize += info.Size()
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Analyze packages
	fh.analyzePackages(project)

	// Cache the project structure
	fh.mu.Lock()
	fh.projects[rootPath] = project
	fh.mu.Unlock()

	return project, nil
}

// BatchRead reads multiple files concurrently
func (fh *FileHandler) BatchRead(paths []string) (map[string][]byte, error) {
	if !fh.config.ParallelReads || len(paths) <= 1 {
		// Read sequentially
		results := make(map[string][]byte)
		for _, path := range paths {
			content, err := fh.ReadFile(path)
			if err != nil {
				return results, err
			}
			results[path] = content
		}
		return results, nil
	}

	// Read in parallel
	type readResult struct {
		path    string
		content []byte
		err     error
	}

	resultChan := make(chan readResult, len(paths))
	var wg sync.WaitGroup

	for _, path := range paths {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			content, err := fh.ReadFile(p)
			resultChan <- readResult{path: p, content: content, err: err}
		}(path)
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	results := make(map[string][]byte)
	for result := range resultChan {
		if result.err != nil {
			return results, result.err
		}
		results[result.path] = result.content
	}

	return results, nil
}

// BatchWrite writes multiple files concurrently
func (fh *FileHandler) BatchWrite(files map[string][]byte, options FileOptions) error {
	if !fh.config.ParallelWrites || len(files) <= 1 {
		// Write sequentially
		for path, content := range files {
			if err := fh.WriteFile(path, content, options); err != nil {
				return err
			}
		}
		return nil
	}

	// Write in parallel
	type writeResult struct {
		path string
		err  error
	}

	resultChan := make(chan writeResult, len(files))
	var wg sync.WaitGroup

	for path, content := range files {
		wg.Add(1)
		go func(p string, c []byte) {
			defer wg.Done()
			err := fh.WriteFile(p, c, options)
			resultChan <- writeResult{path: p, err: err}
		}(path, content)
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	for result := range resultChan {
		if result.err != nil {
			return result.err
		}
	}

	return nil
}

// Operation implementations

func (fh *FileHandler) readOperation(op FileOperation) OperationResult {
	content, err := os.ReadFile(op.Path)
	if err != nil {
		return OperationResult{
			Success: false,
			Error:   err,
		}
	}

	atomic.AddInt64(&fh.reads, 1)
	atomic.AddInt64(&fh.bytesRead, int64(len(content)))

	// Cache if enabled
	if fh.config.EnableCache {
		fh.addToCache(op.Path, content)
	}

	return OperationResult{
		Success: true,
		Data:    content,
	}
}

func (fh *FileHandler) writeOperation(op FileOperation) OperationResult {
	// Create directories if needed
	if op.Options.CreateDirs {
		dir := filepath.Dir(op.Path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return OperationResult{
				Success: false,
				Error:   err,
			}
		}
	}

	// Backup if needed
	if op.Options.Backup && fh.config.BackupBeforeWrite {
		if _, err := os.Stat(op.Path); err == nil {
			backupPath := op.Path + ".backup"
			if err := fh.copyFile(op.Path, backupPath); err != nil {
				return OperationResult{
					Success: false,
					Error:   fmt.Errorf("backup failed: %w", err),
				}
			}
		}
	}

	// Write file
	var err error
	if op.Options.Atomic && fh.config.AtomicWrites {
		err = fh.atomicWrite(op.Path, op.Content, op.Options.Permissions)
	} else {
		perm := op.Options.Permissions
		if perm == 0 {
			perm = 0644
		}
		err = os.WriteFile(op.Path, op.Content, perm)
	}

	if err != nil {
		return OperationResult{
			Success: false,
			Error:   err,
		}
	}

	atomic.AddInt64(&fh.writes, 1)
	atomic.AddInt64(&fh.bytesWritten, int64(len(op.Content)))

	// Update cache
	if fh.config.EnableCache {
		fh.addToCache(op.Path, op.Content)
	}

	return OperationResult{
		Success: true,
	}
}

func (fh *FileHandler) deleteOperation(op FileOperation) OperationResult {
	err := os.Remove(op.Path)
	if err != nil {
		return OperationResult{
			Success: false,
			Error:   err,
		}
	}

	atomic.AddInt64(&fh.deletes, 1)

	return OperationResult{
		Success: true,
	}
}

func (fh *FileHandler) copyOperation(op FileOperation) OperationResult {
	dst := string(op.Content)
	err := fh.copyFile(op.Path, dst)
	if err != nil {
		return OperationResult{
			Success: false,
			Error:   err,
		}
	}

	return OperationResult{
		Success: true,
	}
}

func (fh *FileHandler) moveOperation(op FileOperation) OperationResult {
	dst := string(op.Content)
	err := os.Rename(op.Path, dst)
	if err != nil {
		return OperationResult{
			Success: false,
			Error:   err,
		}
	}

	// Update cache
	if fh.config.EnableCache {
		fh.removeFromCache(op.Path)
	}

	return OperationResult{
		Success: true,
	}
}

func (fh *FileHandler) mkdirOperation(op FileOperation) OperationResult {
	perm := op.Options.Permissions
	if perm == 0 {
		perm = 0755
	}

	err := os.MkdirAll(op.Path, perm)
	if err != nil {
		return OperationResult{
			Success: false,
			Error:   err,
		}
	}

	return OperationResult{
		Success: true,
	}
}

func (fh *FileHandler) listOperation(op FileOperation) OperationResult {
	entries, err := os.ReadDir(op.Path)
	if err != nil {
		return OperationResult{
			Success: false,
			Error:   err,
		}
	}

	var names []string
	for _, entry := range entries {
		names = append(names, entry.Name())
	}

	return OperationResult{
		Success: true,
		Data:    []byte(strings.Join(names, "\n")),
	}
}

func (fh *FileHandler) statOperation(op FileOperation) OperationResult {
	info, err := os.Stat(op.Path)
	if err != nil {
		return OperationResult{
			Success: false,
			Error:   err,
		}
	}

	fileInfo := &FileMetadata{
		Path:    op.Path,
		Name:    info.Name(),
		Size:    info.Size(),
		ModTime: info.ModTime(),
	}

	return OperationResult{
		Success: true,
		Info:    fileInfo,
	}
}

// Helper methods

func (fh *FileHandler) shouldIgnore(path string) bool {
	for _, pattern := range fh.config.IgnorePatterns {
		matched, _ := filepath.Match(pattern, filepath.Base(path))
		if matched {
			return true
		}
		if strings.Contains(path, pattern) {
			return true
		}
	}
	return false
}

func (fh *FileHandler) analyzePackages(project *ProjectStructure) {
	packages := make(map[string]*PackageInfo)

	for path, file := range project.Files {
		if file.Language == "go" || file.Language == "gofa" {
			dir := filepath.Dir(path)
			pkg, exists := packages[dir]
			if !exists {
				pkg = &PackageInfo{
					Path:       dir,
					Name:       filepath.Base(dir),
					ImportPath: strings.TrimPrefix(dir, project.RootPath+"/"),
				}
				packages[dir] = pkg
			}

			if file.Language == "go" {
				if strings.HasSuffix(file.Name, "_test.go") {
					pkg.TestFiles = append(pkg.TestFiles, path)
				} else {
					pkg.GoFiles = append(pkg.GoFiles, path)
				}
			} else {
				pkg.GofaFiles = append(pkg.GofaFiles, path)
			}
		}
	}

	project.Packages = packages
}

func (fh *FileHandler) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

func (fh *FileHandler) atomicWrite(path string, content []byte, perm fs.FileMode) error {
	if perm == 0 {
		perm = 0644
	}

	// Write to temporary file
	tmpFile := path + ".tmp"
	if err := os.WriteFile(tmpFile, content, perm); err != nil {
		return err
	}

	// Atomic rename
	return os.Rename(tmpFile, path)
}

// Cache management

func (fh *FileHandler) getFromCache(path string) *CachedFile {
	fh.mu.RLock()
	defer fh.mu.RUnlock()

	cached, exists := fh.fileCache[path]
	if !exists {
		return nil
	}

	// Check TTL
	if fh.config.CacheTTL > 0 && time.Since(cached.CachedAt) > fh.config.CacheTTL {
		return nil
	}

	atomic.AddInt64(&cached.AccessCount, 1)
	return cached
}

func (fh *FileHandler) addToCache(path string, content []byte) {
	fh.mu.Lock()
	defer fh.mu.Unlock()

	// Calculate hash
	hash := sha256.Sum256(content)

	cached := &CachedFile{
		Path:     path,
		Content:  content,
		Hash:     hex.EncodeToString(hash[:]),
		Size:     int64(len(content)),
		CachedAt: time.Now(),
	}

	fh.fileCache[path] = cached
}

func (fh *FileHandler) removeFromCache(path string) {
	fh.mu.Lock()
	defer fh.mu.Unlock()

	delete(fh.fileCache, path)
}

func (fh *FileHandler) cacheCleaner() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			fh.cleanCache()
		case <-fh.ctx.Done():
			return
		}
	}
}

func (fh *FileHandler) cleanCache() {
	fh.mu.Lock()
	defer fh.mu.Unlock()

	now := time.Now()
	var totalSize int64
	var toDelete []string

	// Calculate total size and find expired entries
	for path, cached := range fh.fileCache {
		totalSize += cached.Size
		if fh.config.CacheTTL > 0 && now.Sub(cached.CachedAt) > fh.config.CacheTTL {
			toDelete = append(toDelete, path)
		}
	}

	// Delete expired entries
	for _, path := range toDelete {
		delete(fh.fileCache, path)
	}

	// Check size limit
	if fh.config.MaxCacheSize > 0 && totalSize > fh.config.MaxCacheSize {
		// Evict least recently used
		// TODO: Implement LRU eviction
	}
}

// GetStatistics returns file handler statistics
func (fh *FileHandler) GetStatistics() map[string]interface{} {
	fh.mu.RLock()
	cacheSize := len(fh.fileCache)
	projectCount := len(fh.projects)
	var cacheBytes int64
	for _, cached := range fh.fileCache {
		cacheBytes += cached.Size
	}
	fh.mu.RUnlock()

	hits := atomic.LoadInt64(&fh.cacheHits)
	misses := atomic.LoadInt64(&fh.cacheMisses)
	hitRate := float64(0)
	if total := hits + misses; total > 0 {
		hitRate = float64(hits) * 100.0 / float64(total)
	}

	return map[string]interface{}{
		"reads":           atomic.LoadInt64(&fh.reads),
		"writes":          atomic.LoadInt64(&fh.writes),
		"deletes":         atomic.LoadInt64(&fh.deletes),
		"bytes_read":      atomic.LoadInt64(&fh.bytesRead),
		"bytes_written":   atomic.LoadInt64(&fh.bytesWritten),
		"cache_hits":      hits,
		"cache_misses":    misses,
		"cache_hit_rate":  hitRate,
		"cache_size":      cacheSize,
		"cache_bytes":     cacheBytes,
		"project_count":   projectCount,
	}
}

// Shutdown shuts down the file handler
func (fh *FileHandler) Shutdown() error {
	fh.cancel()
	close(fh.operations)
	fh.wg.Wait()
	return nil
}

// WatchFile watches a file for changes
func (fh *FileHandler) WatchFile(path string, callback func(FileMetadata)) error {
	if !fh.config.WatchForChanges {
		return fmt.Errorf("file watching is disabled")
	}

	// TODO: Implement file watching
	return fmt.Errorf("file watching not yet implemented")
}

// CreateProject creates a new project structure
func (fh *FileHandler) CreateProject(rootPath string, template string) error {
	// Create project directories
	dirs := []string{
		filepath.Join(rootPath, "cmd"),
		filepath.Join(rootPath, "internal"),
		filepath.Join(rootPath, "pkg"),
		filepath.Join(rootPath, "api"),
		filepath.Join(rootPath, "configs"),
		filepath.Join(rootPath, "scripts"),
		filepath.Join(rootPath, "tests"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	// Create basic files
	files := map[string][]byte{
		filepath.Join(rootPath, "go.mod"):       []byte(fmt.Sprintf("module %s\n\ngo 1.21\n", filepath.Base(rootPath))),
		filepath.Join(rootPath, "README.md"):    []byte(fmt.Sprintf("# %s\n\nProject description here.\n", filepath.Base(rootPath))),
		filepath.Join(rootPath, ".gitignore"):   []byte("*.exe\n*.dll\n*.so\n*.dylib\n*.test\n*.out\nvendor/\n"),
		filepath.Join(rootPath, "Makefile"):     []byte("build:\n\tgo build -o bin/app ./cmd/...\n\ntest:\n\tgo test ./...\n"),
	}

	options := FileOptions{
		CreateDirs: true,
		Overwrite:  false,
	}

	return fh.BatchWrite(files, options)
}
