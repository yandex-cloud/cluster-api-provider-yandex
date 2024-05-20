package loadbalancers

import "context"

// Reconcile reconciles machine instance.
func (s *Service) Reconcile(ctx context.Context) error {
	return nil
}

// Delete deletes machine instance.
func (s *Service) Delete(ctx context.Context) (bool, error) {
	return true, nil
}
