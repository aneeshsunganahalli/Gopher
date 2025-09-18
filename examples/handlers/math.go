package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"math"

	"github.com/aneeshsunganahalli/Gopher/pkg/types"
	"go.uber.org/zap"
)

// MathJobHandler handles mathematical computation jobs
type MathJobHandler struct {
	logger *zap.Logger
}

type MathPayload  struct {
	Operation string `json:"operation"`
	Number int64 `json:"number"`
	Precision int `json:"precision,omitempty"`
}

func NewMathJobHandler(logger *zap.Logger) *MathJobHandler {
	return &MathJobHandler{logger: logger}
}

func (h *MathJobHandler) Type() string {
	return "math"
}

func (h *MathJobHandler) Description() string {
	return "Performs mathematical computations (fibonacci, prime checking, factorial)"
}

func (h *MathJobHandler) Handle(ctx context.Context, job *types.Job) error {
	// Parse payload
	var payload MathPayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return fmt.Errorf("invalid math payload: %w", err)
	}
	
	// Validate required fields
	if payload.Operation == "" {
		return fmt.Errorf("math operation cannot be empty")
	}
	if payload.Number < 0 {
		return fmt.Errorf("number cannot be negative")
	}
	
	h.logger.Info("Starting math computation",
		zap.String("job_id", job.ID),
		zap.String("operation", payload.Operation),
		zap.Int64("number", payload.Number),
	)
	
	var result interface{}
	var err error
	
	// Perform computation based on operation type
	switch payload.Operation {
	case "fibonacci":
		result, err = h.fibonacci(ctx, payload.Number)
	case "prime":
		result, err = h.isPrime(ctx, payload.Number)
	case "factorial":
		result, err = h.factorial(ctx, payload.Number)
	default:
		return fmt.Errorf("unsupported operation: %s", payload.Operation)
	}
	
	if err != nil {
		return fmt.Errorf("computation failed: %w", err)
	}
	
	h.logger.Info("Math computation completed",
		zap.String("job_id", job.ID),
		zap.String("operation", payload.Operation),
		zap.Int64("number", payload.Number),
		zap.Any("result", result),
	)
	
	return nil
}

func (h *MathJobHandler) fibonacci(ctx context.Context, n int64) (int64, error) {
	if n <= 1 {
		return n, nil
	}
	
	// Use iterative approach for better performance
	var a, b int64 = 0, 1
	for i := int64(2); i <= n; i++ {
		// Check for context cancellation periodically
		if i%1000000 == 0 {
			select {
			case <-ctx.Done():
				return 0, ctx.Err()
			default:
			}
		}
		
		a, b = b, a+b
	}
	
	return b, nil
}

func (h *MathJobHandler) isPrime(ctx context.Context, n int64) (bool, error) {
	if n < 2 {
		return false, nil
	}
	if n == 2 {
		return true, nil
	}
	if n%2 == 0 {
		return false, nil
	}
	
	// Check odd divisors up to sqrt(n)
	sqrt := int64(math.Sqrt(float64(n)))
	for i := int64(3); i <= sqrt; i += 2 {
		// Check for context cancellation periodically
		if i%100000 == 0 {
			select {
			case <-ctx.Done():
				return false, ctx.Err()
			default:
			}
		}
		
		if n%i == 0 {
			return false, nil
		}
	}
	
	return true, nil
}

func (h *MathJobHandler) factorial(ctx context.Context, n int64) (int64, error) {
	if n < 0 {
		return 0, fmt.Errorf("factorial of negative number is undefined")
	}
	if n > 20 {
		return 0, fmt.Errorf("factorial too large (n > 20), would overflow")
	}
	
	result := int64(1)
	for i := int64(2); i <= n; i++ {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		default:
		}
		
		result *= i
	}
	
	return result, nil
}