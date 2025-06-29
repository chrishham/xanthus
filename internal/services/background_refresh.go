package services

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// BackgroundRefreshService handles background version updates
type BackgroundRefreshService struct {
	versionService EnhancedVersionService
	refreshQueue   chan refreshRequest
	workers        int
	stopChan       chan struct{}
	wg             sync.WaitGroup
	ctx            context.Context
	cancel         context.CancelFunc
	running        bool
	mutex          sync.RWMutex
}

type refreshRequest struct {
	appID    string
	priority RefreshPriority
	callback func(appID string, version string, err error)
}

// RefreshPriority defines the priority of refresh requests
type RefreshPriority int

const (
	PriorityLow RefreshPriority = iota
	PriorityNormal
	PriorityHigh
	PriorityUrgent
)

// BackgroundRefreshConfig configures the background refresh service
type BackgroundRefreshConfig struct {
	Workers         int
	QueueSize       int
	RefreshInterval time.Duration
	RetryAttempts   int
	RetryDelay      time.Duration
}

// NewBackgroundRefreshService creates a new background refresh service
func NewBackgroundRefreshService(versionService EnhancedVersionService, config BackgroundRefreshConfig) *BackgroundRefreshService {
	ctx, cancel := context.WithCancel(context.Background())

	return &BackgroundRefreshService{
		versionService: versionService,
		refreshQueue:   make(chan refreshRequest, config.QueueSize),
		workers:        config.Workers,
		stopChan:       make(chan struct{}),
		ctx:            ctx,
		cancel:         cancel,
	}
}

// Start begins the background refresh service
func (b *BackgroundRefreshService) Start() error {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if b.running {
		return nil // Already running
	}

	log.Printf("Starting background refresh service with %d workers", b.workers)

	// Start worker goroutines
	for i := 0; i < b.workers; i++ {
		b.wg.Add(1)
		go b.worker(i)
	}

	b.running = true
	return nil
}

// Stop gracefully shuts down the background refresh service
func (b *BackgroundRefreshService) Stop() error {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if !b.running {
		return nil // Already stopped
	}

	log.Println("Stopping background refresh service...")

	// Cancel context and close stop channel
	b.cancel()
	close(b.stopChan)

	// Wait for all workers to finish
	b.wg.Wait()

	// Close refresh queue
	close(b.refreshQueue)

	b.running = false
	log.Println("Background refresh service stopped")
	return nil
}

// QueueRefresh adds a refresh request to the background queue
func (b *BackgroundRefreshService) QueueRefresh(appID string, priority RefreshPriority, callback func(string, string, error)) {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	if !b.running {
		log.Printf("Background refresh service not running, skipping refresh for %s", appID)
		if callback != nil {
			callback(appID, "", fmt.Errorf("background refresh service not running"))
		}
		return
	}

	select {
	case b.refreshQueue <- refreshRequest{
		appID:    appID,
		priority: priority,
		callback: callback,
	}:
		log.Printf("Queued background refresh for %s with priority %d", appID, priority)
	default:
		log.Printf("Refresh queue full, dropping request for %s", appID)
		if callback != nil {
			callback(appID, "", fmt.Errorf("refresh queue full"))
		}
	}
}

// IsRunning returns whether the service is currently running
func (b *BackgroundRefreshService) IsRunning() bool {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	return b.running
}

// worker processes refresh requests from the queue
func (b *BackgroundRefreshService) worker(workerID int) {
	defer b.wg.Done()

	log.Printf("Background refresh worker %d started", workerID)

	for {
		select {
		case <-b.ctx.Done():
			log.Printf("Background refresh worker %d stopping due to context cancellation", workerID)
			return
		case <-b.stopChan:
			log.Printf("Background refresh worker %d stopping due to stop signal", workerID)
			return
		case request, ok := <-b.refreshQueue:
			if !ok {
				log.Printf("Background refresh worker %d stopping due to closed queue", workerID)
				return
			}

			b.processRefreshRequest(workerID, request)
		}
	}
}

// processRefreshRequest handles a single refresh request
func (b *BackgroundRefreshService) processRefreshRequest(workerID int, request refreshRequest) {
	log.Printf("Worker %d processing refresh for %s", workerID, request.appID)

	start := time.Now()
	version, err := b.versionService.RefreshVersionFromSource(request.appID)
	duration := time.Since(start)

	if err != nil {
		log.Printf("Worker %d failed to refresh %s after %v: %v", workerID, request.appID, duration, err)
	} else {
		log.Printf("Worker %d successfully refreshed %s to version %s in %v", workerID, request.appID, version, duration)
	}

	// Call callback if provided
	if request.callback != nil {
		request.callback(request.appID, version, err)
	}
}

// PeriodicRefreshManager handles scheduled background refreshes
type PeriodicRefreshManager struct {
	backgroundService *BackgroundRefreshService
	catalogService    ApplicationCatalog
	ticker            *time.Ticker
	stopChan          chan struct{}
	running           bool
	mutex             sync.RWMutex
}

// NewPeriodicRefreshManager creates a new periodic refresh manager
func NewPeriodicRefreshManager(backgroundService *BackgroundRefreshService, catalogService ApplicationCatalog, interval time.Duration) *PeriodicRefreshManager {
	return &PeriodicRefreshManager{
		backgroundService: backgroundService,
		catalogService:    catalogService,
		ticker:            time.NewTicker(interval),
		stopChan:          make(chan struct{}),
	}
}

// Start begins periodic refreshing of all applications
func (p *PeriodicRefreshManager) Start() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.running {
		return
	}

	log.Println("Starting periodic refresh manager")
	p.running = true

	go p.refreshLoop()
}

// Stop stops the periodic refresh manager
func (p *PeriodicRefreshManager) Stop() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if !p.running {
		return
	}

	log.Println("Stopping periodic refresh manager")
	p.ticker.Stop()
	close(p.stopChan)
	p.running = false
}

// refreshLoop runs the periodic refresh cycle
func (p *PeriodicRefreshManager) refreshLoop() {
	for {
		select {
		case <-p.stopChan:
			return
		case <-p.ticker.C:
			p.refreshAllApplications()
		}
	}
}

// refreshAllApplications queues refresh requests for all applications
func (p *PeriodicRefreshManager) refreshAllApplications() {
	applications := p.catalogService.GetApplications()

	log.Printf("Queuing periodic refresh for %d applications", len(applications))

	for _, app := range applications {
		p.backgroundService.QueueRefresh(app.ID, PriorityLow, func(appID, version string, err error) {
			if err != nil {
				log.Printf("Periodic refresh failed for %s: %v", appID, err)
			} else {
				log.Printf("Periodic refresh completed for %s: %s", appID, version)
			}
		})
	}
}
