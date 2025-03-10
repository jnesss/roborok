package report

import (
	"log"
	"time"
)

// Reporter handles sending stats and screenshots to the backend
type Reporter struct {
	Endpoint    string
	stopChan    chan struct{}
	reportQueue []ReportItem
}

// ReportItem represents an item in the report queue
type ReportItem struct {
	InstanceID     string
	ScreenshotPath string
	Metadata       map[string]interface{}
	Timestamp      time.Time
}

// NewReporter creates a new reporter
func NewReporter(endpoint string) *Reporter {
	return &Reporter{
		Endpoint:    endpoint,
		stopChan:    make(chan struct{}),
		reportQueue: make([]ReportItem, 0),
	}
}

// Start starts the reporter background process
func (r *Reporter) Start() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	log.Println("Reporter started")

	for {
		select {
		case <-ticker.C:
			// This is a placeholder - in the future this will send data to the backend
			if len(r.reportQueue) > 0 {
				log.Printf("Would send %d reports to backend", len(r.reportQueue))
				// Clear the queue after reporting
				r.reportQueue = r.reportQueue[:0]
			}
		case <-r.stopChan:
			log.Println("Reporter stopped")
			return
		}
	}
}

// Stop stops the reporter
func (r *Reporter) Stop() {
	close(r.stopChan)
}

// ReportScreenshot adds a screenshot report to the queue
func (r *Reporter) ReportScreenshot(instanceID, screenshotPath string, metadata map[string]interface{}) {
	// Add to queue
	r.reportQueue = append(r.reportQueue, ReportItem{
		InstanceID:     instanceID,
		ScreenshotPath: screenshotPath,
		Metadata:       metadata,
		Timestamp:      time.Now(),
	})

	// Log for now
	log.Printf("Screenshot reported for instance %s: %s", instanceID, screenshotPath)
}

// ReportEvent adds an event report to the queue
func (r *Reporter) ReportEvent(instanceID string, eventType string, metadata map[string]interface{}) {
	// Add to queue
	r.reportQueue = append(r.reportQueue, ReportItem{
		InstanceID:     instanceID,
		ScreenshotPath: "",
		Metadata: map[string]interface{}{
			"event_type": eventType,
			"data":       metadata,
		},
		Timestamp: time.Now(),
	})

	// Log for now
	log.Printf("Event reported for instance %s: %s", instanceID, eventType)
}
