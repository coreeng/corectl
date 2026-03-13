package tenant

import (
	"errors"
	"fmt"

	coretnt "github.com/coreeng/core-platform/pkg/tenant"
)

// ValidateNewTenant validates a tenant being introduced into an existing tenant set.
//
// It is intentionally tolerant of pre-existing unrelated validation errors in the repo.
// The returned error only contains issues related to the provided tenant.
func ValidateNewTenant(existing []coretnt.Tenant, t *coretnt.Tenant) error {
	if t == nil {
		return fmt.Errorf("tenant is nil")
	}

	// Always validate the new tenant's fields.
	fieldErrs := t.Validate()
	if len(fieldErrs) > 0 {
		errList := make([]error, 0, len(fieldErrs))
		for _, fe := range fieldErrs {
			errList = append(errList, fmt.Errorf("%s", fe.Error()))
		}
		return errors.Join(errList...)
	}

	// Core-platform ValidateTenants includes some global checks that do not identify the tenant.
	// Enforce what we can for the new tenant explicitly.
	if t.Kind == "OrgUnit" {
		var errs []error
		if t.AdminGroup == "" {
			errs = append(errs, fmt.Errorf("admin group must be present for an org unit"))
		}
		if t.ReadOnlyGroup == "" {
			errs = append(errs, fmt.Errorf("read only group must be present for an org unit"))
		}
		if len(errs) > 0 {
			return errors.Join(errs...)
		}
	}

	// Run cross-tenant validation and keep only errors related to the new tenant.
	tenantMap := map[string]*coretnt.Tenant{t.Name: t}
	for i := range existing {
		et := existing[i]
		tenantMap[et.Name] = &et
	}

	res := coretnt.ValidateTenants(tenantMap)
	if len(res.Errors) == 0 {
		return nil
	}

	filtered := make([]error, 0)
	for _, e := range res.Errors {
		var tr coretnt.TenantRelatedError
		if errors.As(e, &tr) {
			if tr.IsRelatedToTenant(t) {
				filtered = append(filtered, e)
			}
			continue
		}
		var gd *coretnt.GroupDuplicationError
		if errors.As(e, &gd) {
			if gd.IsRelatedToTenant(t) {
				filtered = append(filtered, e)
			}
			continue
		}
	}

	if len(filtered) == 0 {
		return nil
	}
	return errors.Join(filtered...)
}
