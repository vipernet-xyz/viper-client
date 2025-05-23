package cron

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/illegalcall/viper-client/internal/performancemeasurement"
	"github.com/illegalcall/viper-client/internal/servicerdiscovery"
	"github.com/illegalcall/viper-client/internal/utils"
	"github.com/robfig/cron/v3"
)

// CronJobOrchestrator manages scheduled tasks for servicer discovery and performance measurement.
type CronJobOrchestrator struct {
	config             *utils.Config // Full config to pass down MonitoringConfig part
	db                 *sql.DB
	discoveryService   *servicerdiscovery.ServicerDiscoveryService
	measurementService *performancemeasurement.PerformanceMeasurerService
	cronScheduler      *cron.Cron
	jobCtx             context.Context
	jobCancel          context.CancelFunc
}

// NewCronJobOrchestrator creates a new instance of CronJobOrchestrator.
func NewCronJobOrchestrator(cfg *utils.Config, dbConn *sql.DB) *CronJobOrchestrator {
	discoverySvc := servicerdiscovery.NewServicerDiscoveryService(&cfg.Monitoring, dbConn)
	measurementSvc := performancemeasurement.NewPerformanceMeasurerService(&cfg.Monitoring, dbConn)

	// Create a cancellable context for the jobs
	jobCtx, jobCancel := context.WithCancel(context.Background())

	return &CronJobOrchestrator{
		config:             cfg,
		db:                 dbConn,
		discoveryService:   discoverySvc,
		measurementService: measurementSvc,
		cronScheduler:      cron.New(cron.WithSeconds()), // Using WithSeconds() for flexibility as per spec
		jobCtx:             jobCtx,
		jobCancel:          jobCancel,
	}
}

// Start initializes and starts the cron jobs.
func (o *CronJobOrchestrator) Start() error {
	log.Println("[CronOrchestrator] Starting Cron Job Orchestrator...")
	if o.config.Monitoring.CronSchedule == "" {
		log.Println("[CronOrchestrator] MONITORING_CRON_SCHEDULE is not set. Orchestrator will not schedule jobs.")
		return nil // Or return an error if this should be fatal, as per spec
	}

	// Main monitoring job
	_, err := o.cronScheduler.AddFunc(o.config.Monitoring.CronSchedule, o.runMonitoringCycle)
	if err != nil {
		log.Printf("[CronOrchestrator] Error scheduling monitoring cycle: %v", err)
		return err
	}
	log.Printf("[CronOrchestrator] Scheduled monitoring cycle with schedule: %s", o.config.Monitoring.CronSchedule)

	// TODO: Add other jobs if needed, e.g., database cleanup based on o.config.Monitoring.CleanupInterval

	o.cronScheduler.Start()
	log.Println("[CronOrchestrator] Cron scheduler started.")
	return nil
}

// Stop gracefully stops the cron scheduler and cancels any running jobs.
func (o *CronJobOrchestrator) Stop() {
	log.Println("[CronOrchestrator] Stopping Cron Job Orchestrator...")
	if o.cronScheduler != nil {
		// cron.Stop() returns a context that is done when all running jobs complete.
		stopCtx := o.cronScheduler.Stop()
		log.Println("[CronOrchestrator] Cron scheduler stopping. Waiting for running jobs to complete...")
		select {
		case <-stopCtx.Done():
			log.Println("[CronOrchestrator] All cron jobs finished.")
		case <-time.After(30 * time.Second): // Timeout for graceful shutdown
			log.Println("[CronOrchestrator] Timeout waiting for cron jobs to finish. Forcing stop.")
		}
	}
	o.jobCancel() // Signal all orchestrated jobs (via jobCtx) to cancel
	log.Println("[CronOrchestrator] Cron Job Orchestrator stopped.")
}

func (o *CronJobOrchestrator) runMonitoringCycle() {
	log.Println("[CronOrchestrator] Starting new monitoring cycle...")
	startTime := time.Now()

	// Create a new context for this specific run that respects the orchestrator's jobCtx
	// and has its own timeout for the entire cycle.
	runCtx, runCancel := context.WithTimeout(o.jobCtx, 15*time.Minute) // Example: 15 min timeout for the whole cycle
	defer runCancel()

	// 1. Discover and Register Servicers
	log.Println("[CronOrchestrator] Step 1: Running Servicer Discovery and Registration...")
	if err := o.discoveryService.DiscoverAndRegisterServicers(runCtx); err != nil {
		// Check if the error is due to context cancellation (orchestrator shutting down)
		if runCtx.Err() == context.Canceled {
			log.Println("[CronOrchestrator] Servicer discovery cancelled during shutdown.")
			return // Exit if main context (jobCtx) was cancelled
		} else if runCtx.Err() == context.DeadlineExceeded {
			log.Println("[CronOrchestrator] Servicer discovery timed out for this cycle.")
			// Log and continue to performance measurement or return, based on desired behavior
		} else {
			log.Printf("[CronOrchestrator] Error during servicer discovery: %v", err)
		}
		// Decide if cycle should continue if discovery fails. For now, it continues as per spec.
	} else {
		log.Println("[CronOrchestrator] Servicer Discovery and Registration completed.")
	}

	// Check for cancellation or timeout before starting the next long step
	if runCtx.Err() != nil {
		log.Printf("[CronOrchestrator] Monitoring cycle cancelled or timed out before performance measurement. Error: %v", runCtx.Err())
		return
	}

	// 2. Measure Performance
	log.Println("[CronOrchestrator] Step 2: Running Performance Measurement...")
	if err := o.measurementService.MeasurePerformance(runCtx); err != nil {
		if runCtx.Err() == context.Canceled {
			log.Println("[CronOrchestrator] Performance measurement cancelled during shutdown.")
			return // Exit if main context (jobCtx) was cancelled
		} else if runCtx.Err() == context.DeadlineExceeded {
			log.Println("[CronOrchestrator] Performance measurement timed out for this cycle.")
		} else {
			log.Printf("[CronOrchestrator] Error during performance measurement: %v", err)
		}
	} else {
		log.Println("[CronOrchestrator] Performance Measurement completed.")
	}

	duration := time.Since(startTime)
	log.Printf("[CronOrchestrator] Monitoring cycle finished. Duration: %s", duration)
}
