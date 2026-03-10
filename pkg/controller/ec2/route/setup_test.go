package route

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	svcsdk "github.com/aws/aws-sdk-go/service/ec2"
	svcsdkapi "github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	svcapitypes "github.com/crossplane-contrib/provider-aws/apis/ec2/v1alpha1"
)

type mockEC2Client struct {
	svcsdkapi.EC2API
	describeRouteTablesOutput *svcsdk.DescribeRouteTablesOutput
	describeRouteTablesErr    error
}

func (m *mockEC2Client) DescribeRouteTablesWithContext(_ aws.Context, _ *svcsdk.DescribeRouteTablesInput, _ ...request.Option) (*svcsdk.DescribeRouteTablesOutput, error) {
	return m.describeRouteTablesOutput, m.describeRouteTablesErr
}

func routeTableWithRoutes(routes ...*svcsdk.Route) *svcsdk.DescribeRouteTablesOutput {
	return &svcsdk.DescribeRouteTablesOutput{
		RouteTables: []*svcsdk.RouteTable{
			{Routes: routes},
		},
	}
}

func TestFindRouteByDestination(t *testing.T) {
	tests := []struct {
		name    string
		routes  []*svcsdk.Route
		cr      *svcapitypes.Route
		wantErr bool
	}{
		{
			name: "finds IPv4 route",
			routes: []*svcsdk.Route{
				{
					DestinationCidrBlock: aws.String("10.0.0.0/16"),
					Origin:               aws.String(svcsdk.RouteOriginCreateRoute),
					State:                aws.String(svcsdk.RouteStateActive),
				},
			},
			cr: makeRoute(withIPv4CIDR("10.0.0.0/16")),
		},
		{
			name: "finds IPv6 route",
			routes: []*svcsdk.Route{
				{
					DestinationIpv6CidrBlock: aws.String("2604:2be0::/32"),
					Origin:                   aws.String(svcsdk.RouteOriginCreateRoute),
					State:                    aws.String(svcsdk.RouteStateActive),
				},
			},
			cr: makeRoute(withIPv6CIDR("2604:2be0::/32")),
		},
		{
			name: "finds IPv6 route among mixed IPv4 and IPv6 routes",
			routes: []*svcsdk.Route{
				{
					DestinationCidrBlock: aws.String("0.0.0.0/0"),
					Origin:               aws.String(svcsdk.RouteOriginCreateRoute),
					State:                aws.String(svcsdk.RouteStateActive),
				},
				{
					DestinationIpv6CidrBlock: aws.String("2604:2be0::/32"),
					Origin:                   aws.String(svcsdk.RouteOriginCreateRoute),
					State:                    aws.String(svcsdk.RouteStateActive),
				},
			},
			cr: makeRoute(withIPv6CIDR("2604:2be0::/32")),
		},
		{
			name: "skips propagated routes",
			routes: []*svcsdk.Route{
				{
					DestinationCidrBlock: aws.String("10.0.0.0/16"),
					Origin:               aws.String(svcsdk.RouteOriginEnableVgwRoutePropagation),
					State:                aws.String(svcsdk.RouteStateActive),
				},
			},
			cr:      makeRoute(withIPv4CIDR("10.0.0.0/16")),
			wantErr: true,
		},
		{
			name:    "no matching route returns error",
			routes:  []*svcsdk.Route{},
			cr:      makeRoute(withIPv6CIDR("2604:2be0::/32")),
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := &external{
				client: &mockEC2Client{
					describeRouteTablesOutput: routeTableWithRoutes(tc.routes...),
				},
			}

			route, err := e.findRouteByDestination(context.Background(), tc.cr)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if route == nil {
				t.Fatal("expected route, got nil")
			}
		})
	}
}

func TestObserver(t *testing.T) {
	tests := []struct {
		name       string
		routes     []*svcsdk.Route
		cr         *svcapitypes.Route
		wantExists bool
	}{
		{
			name: "IPv4 route is observed as existing",
			routes: []*svcsdk.Route{
				{
					DestinationCidrBlock: aws.String("10.0.0.0/16"),
					Origin:               aws.String(svcsdk.RouteOriginCreateRoute),
					State:                aws.String(svcsdk.RouteStateActive),
				},
			},
			cr:         makeRoute(withIPv4CIDR("10.0.0.0/16")),
			wantExists: true,
		},
		{
			name: "IPv6 route is observed as existing",
			routes: []*svcsdk.Route{
				{
					DestinationIpv6CidrBlock: aws.String("2604:2be0::/32"),
					Origin:                   aws.String(svcsdk.RouteOriginCreateRoute),
					State:                    aws.String(svcsdk.RouteStateActive),
				},
			},
			cr:         makeRoute(withIPv6CIDR("2604:2be0::/32")),
			wantExists: true,
		},
		{
			name:       "missing route is observed as not existing",
			routes:     []*svcsdk.Route{},
			cr:         makeRoute(withIPv6CIDR("2604:2be0::/32")),
			wantExists: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := &external{
				client: &mockEC2Client{
					describeRouteTablesOutput: routeTableWithRoutes(tc.routes...),
				},
			}
			e.observe = e.observer

			obs, err := e.Observe(context.Background(), tc.cr)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if obs.ResourceExists != tc.wantExists {
				t.Errorf("ResourceExists = %v, want %v", obs.ResourceExists, tc.wantExists)
			}
		})
	}
}

type routeModifier func(*svcapitypes.Route)

func makeRoute(mods ...routeModifier) *svcapitypes.Route {
	cr := &svcapitypes.Route{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{},
		},
		Spec: svcapitypes.RouteSpec{
			ForProvider: svcapitypes.RouteParameters{
				CustomRouteParameters: svcapitypes.CustomRouteParameters{
					RouteTableID: aws.String("rtb-test"),
				},
			},
		},
	}
	meta.SetExternalName(cr, "test-route")
	for _, m := range mods {
		m(cr)
	}
	return cr
}

func withIPv4CIDR(cidr string) routeModifier {
	return func(r *svcapitypes.Route) {
		r.Spec.ForProvider.DestinationCIDRBlock = aws.String(cidr)
	}
}

func withIPv6CIDR(cidr string) routeModifier {
	return func(r *svcapitypes.Route) {
		r.Spec.ForProvider.DestinationIPv6CIDRBlock = aws.String(cidr)
	}
}

var _ managed.ExternalClient = &external{}
