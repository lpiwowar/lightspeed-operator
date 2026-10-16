package controller

import "testing"

func TestClusterRBACNamesAreScoped(t *testing.T) {
	const namespace = "openstack-lightspeed"
	const scopeID = "fab9e71336d4"

	if got := AccessClusterRoleName(namespace); got != "lightspeed-access-"+scopeID {
		t.Errorf("unexpected access role name %q", got)
	}
	if got := sarClusterRoleName(namespace); got != "lightspeed-sar-role-"+scopeID {
		t.Errorf("unexpected SAR role name %q", got)
	}
	if got := sarClusterRoleBindingName(namespace); got != "lightspeed-sar-binding-"+scopeID {
		t.Errorf("unexpected SAR binding name %q", got)
	}
	if AccessClusterRoleName("other") == AccessClusterRoleName(namespace) {
		t.Error("different namespaces must use different access roles")
	}
}
