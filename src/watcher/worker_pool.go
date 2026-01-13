package watcher

import (
	"log/slog"
	"sync"

	"gaa/file-organizer/src/config"
	"gaa/file-organizer/src/processor"
)

// Job representa uma tarefa de processamento de arquivo
type Job struct {
	FilePath string
	Rules    []config.Rule
}

// WorkerPool gerencia um pool de goroutines para processar arquivos
type WorkerPool struct {
	jobsCh  chan Job
	workers int
	logger  *slog.Logger
	wg      sync.WaitGroup
	stopCh  chan struct{}
}

// NewWorkerPool cria um novo worker pool
func NewWorkerPool(workers int, logger *slog.Logger) *WorkerPool {
	// Buffer = 2x workers para evitar bloqueio
	jobsCh := make(chan Job, workers*2)

	return &WorkerPool{
		jobsCh:  jobsCh,
		workers: workers,
		logger:  logger,
		stopCh:  make(chan struct{}),
	}
}

// Start inicia todos os workers do pool
func (wp *WorkerPool) Start() {
	wp.logger.Info("Starting worker pool", "workers", wp.workers)

	for i := 0; i < wp.workers; i++ {
		wp.wg.Add(1)
		go wp.worker(i)
	}
}

// worker é a goroutine que processa jobs
func (wp *WorkerPool) worker(id int) {
	defer wp.wg.Done()

	wp.logger.Debug("Worker started", "worker_id", id)

	for {
		select {
		case job, ok := <-wp.jobsCh:
			if !ok {
				wp.logger.Debug("Worker stopping (channel closed)", "worker_id", id)
				return // Canal fechado - shutdown
			}

			// Processar job com error recovery
			func() {
				defer func() {
					if r := recover(); r != nil {
						wp.logger.Error("Worker panic recovered",
							"worker_id", id,
							"panic", r,
							"file", job.FilePath,
						)
					}
				}()

				wp.logger.Debug("Worker processing file",
					"worker_id", id,
					"file", job.FilePath,
				)

				// Matching + Move
				rule := processor.MatchRule(job.FilePath, job.Rules)
				if rule == nil {
					wp.logger.Debug("No matching rule for file",
						"worker_id", id,
						"file", job.FilePath,
					)
					return
				}

				wp.logger.Info("Worker matched rule",
					"worker_id", id,
					"file", job.FilePath,
					"rule", rule.Name,
				)

				err := processor.MoveFile(
					job.FilePath,
					rule.Destination,
					rule.ConflictStrategy,
					wp.logger,
				)
				if err != nil {
					wp.logger.Error("Worker failed to move file",
						"worker_id", id,
						"file", job.FilePath,
						"error", err,
					)
				} else {
					wp.logger.Info("Worker completed job",
						"worker_id", id,
						"file", job.FilePath,
					)
				}
			}()

		case <-wp.stopCh:
			wp.logger.Debug("Worker stopping (stop signal)", "worker_id", id)
			return
		}
	}
}

// Submit envia um job para o pool
func (wp *WorkerPool) Submit(job Job) {
	select {
	case wp.jobsCh <- job:
		wp.logger.Debug("Job submitted to worker pool", "file", job.FilePath)
	default:
		// Canal cheio - logar warning mas não bloquear
		wp.logger.Warn("Worker pool channel full, job may be delayed", "file", job.FilePath)
		wp.jobsCh <- job // Bloquear até ter espaço
	}
}

// Stop para o worker pool gracefully
func (wp *WorkerPool) Stop() {
	wp.logger.Info("Stopping worker pool")

	// Fechar canal de jobs (não aceitar mais trabalho)
	close(wp.jobsCh)

	// Sinalizar workers para parar
	close(wp.stopCh)

	// Aguardar todos os workers terminarem
	wp.wg.Wait()

	wp.logger.Info("Worker pool stopped")
}
