
package validate

import "fmt"

func MaxSize(b []byte, hard int) error {
	if hard > 0 && len(b) > hard {
		return fmt.Errorf("payload exceeds HARD_MAX: %d > %d", len(b), hard)
	}
	return nil
}
