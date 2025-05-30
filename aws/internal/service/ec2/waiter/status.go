package waiter

import (
	"fmt"
	"log"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/aws-sdk-go-base/tfawserr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	tfec2 "github.com/terraform-providers/terraform-provider-aws/aws/internal/service/ec2"
	"github.com/terraform-providers/terraform-provider-aws/aws/internal/service/ec2/finder"
	tfiam "github.com/terraform-providers/terraform-provider-aws/aws/internal/service/iam"
	"github.com/terraform-providers/terraform-provider-aws/aws/internal/tfresource"
)

const (
	carrierGatewayStateNotFound = "NotFound"
	carrierGatewayStateUnknown  = "Unknown"
)

// CarrierGatewayState fetches the CarrierGateway and its State
func CarrierGatewayState(conn *ec2.EC2, carrierGatewayID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		carrierGateway, err := finder.CarrierGatewayByID(conn, carrierGatewayID)
		if tfawserr.ErrCodeEquals(err, tfec2.ErrCodeInvalidCarrierGatewayIDNotFound) {
			return nil, carrierGatewayStateNotFound, nil
		}
		if err != nil {
			return nil, carrierGatewayStateUnknown, err
		}

		if carrierGateway == nil {
			return nil, carrierGatewayStateNotFound, nil
		}

		state := aws.StringValue(carrierGateway.State)

		if state == ec2.CarrierGatewayStateDeleted {
			return nil, carrierGatewayStateNotFound, nil
		}

		return carrierGateway, state, nil
	}
}

// LocalGatewayRouteTableVpcAssociationState fetches the LocalGatewayRouteTableVpcAssociation and its State
func LocalGatewayRouteTableVpcAssociationState(conn *ec2.EC2, localGatewayRouteTableVpcAssociationID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		input := &ec2.DescribeLocalGatewayRouteTableVpcAssociationsInput{
			LocalGatewayRouteTableVpcAssociationIds: aws.StringSlice([]string{localGatewayRouteTableVpcAssociationID}),
		}

		output, err := conn.DescribeLocalGatewayRouteTableVpcAssociations(input)

		if err != nil {
			return nil, "", err
		}

		var association *ec2.LocalGatewayRouteTableVpcAssociation

		for _, outputAssociation := range output.LocalGatewayRouteTableVpcAssociations {
			if outputAssociation == nil {
				continue
			}

			if aws.StringValue(outputAssociation.LocalGatewayRouteTableVpcAssociationId) == localGatewayRouteTableVpcAssociationID {
				association = outputAssociation
				break
			}
		}

		if association == nil {
			return association, ec2.RouteTableAssociationStateCodeDisassociated, nil
		}

		return association, aws.StringValue(association.State), nil
	}
}

const (
	ClientVpnEndpointStatusNotFound = "NotFound"

	ClientVpnEndpointStatusUnknown = "Unknown"
)

// ClientVpnEndpointStatus fetches the Client VPN endpoint and its Status
func ClientVpnEndpointStatus(conn *ec2.EC2, endpointID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		result, err := conn.DescribeClientVpnEndpoints(&ec2.DescribeClientVpnEndpointsInput{
			ClientVpnEndpointIds: aws.StringSlice([]string{endpointID}),
		})
		if tfawserr.ErrCodeEquals(err, tfec2.ErrCodeClientVpnEndpointIdNotFound) {
			return nil, ClientVpnEndpointStatusNotFound, nil
		}
		if err != nil {
			return nil, ClientVpnEndpointStatusUnknown, err
		}

		if result == nil || len(result.ClientVpnEndpoints) == 0 || result.ClientVpnEndpoints[0] == nil {
			return nil, ClientVpnEndpointStatusNotFound, nil
		}

		endpoint := result.ClientVpnEndpoints[0]
		if endpoint.Status == nil || endpoint.Status.Code == nil {
			return endpoint, ClientVpnEndpointStatusUnknown, nil
		}

		return endpoint, aws.StringValue(endpoint.Status.Code), nil
	}
}

const (
	ClientVpnAuthorizationRuleStatusNotFound = "NotFound"

	ClientVpnAuthorizationRuleStatusUnknown = "Unknown"
)

// ClientVpnAuthorizationRuleStatus fetches the Client VPN authorization rule and its Status
func ClientVpnAuthorizationRuleStatus(conn *ec2.EC2, authorizationRuleID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		result, err := finder.ClientVpnAuthorizationRuleByID(conn, authorizationRuleID)
		if tfawserr.ErrCodeEquals(err, tfec2.ErrCodeClientVpnAuthorizationRuleNotFound) {
			return nil, ClientVpnAuthorizationRuleStatusNotFound, nil
		}
		if err != nil {
			return nil, ClientVpnAuthorizationRuleStatusUnknown, err
		}

		if result == nil || len(result.AuthorizationRules) == 0 || result.AuthorizationRules[0] == nil {
			return nil, ClientVpnAuthorizationRuleStatusNotFound, nil
		}

		if len(result.AuthorizationRules) > 1 {
			return nil, ClientVpnAuthorizationRuleStatusUnknown, fmt.Errorf("internal error: found %d results for Client VPN authorization rule (%s) status, need 1", len(result.AuthorizationRules), authorizationRuleID)
		}

		rule := result.AuthorizationRules[0]
		if rule.Status == nil || rule.Status.Code == nil {
			return rule, ClientVpnAuthorizationRuleStatusUnknown, nil
		}

		return rule, aws.StringValue(rule.Status.Code), nil
	}
}

const (
	ClientVpnNetworkAssociationStatusNotFound = "NotFound"

	ClientVpnNetworkAssociationStatusUnknown = "Unknown"
)

func ClientVpnNetworkAssociationStatus(conn *ec2.EC2, cvnaID string, cvepID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		result, err := conn.DescribeClientVpnTargetNetworks(&ec2.DescribeClientVpnTargetNetworksInput{
			ClientVpnEndpointId: aws.String(cvepID),
			AssociationIds:      []*string{aws.String(cvnaID)},
		})

		if tfawserr.ErrCodeEquals(err, tfec2.ErrCodeClientVpnAssociationIdNotFound) || tfawserr.ErrCodeEquals(err, tfec2.ErrCodeClientVpnEndpointIdNotFound) {
			return nil, ClientVpnNetworkAssociationStatusNotFound, nil
		}
		if err != nil {
			return nil, ClientVpnNetworkAssociationStatusUnknown, err
		}

		if result == nil || len(result.ClientVpnTargetNetworks) == 0 || result.ClientVpnTargetNetworks[0] == nil {
			return nil, ClientVpnNetworkAssociationStatusNotFound, nil
		}

		network := result.ClientVpnTargetNetworks[0]
		if network.Status == nil || network.Status.Code == nil {
			return network, ClientVpnNetworkAssociationStatusUnknown, nil
		}

		return network, aws.StringValue(network.Status.Code), nil
	}
}

const (
	ClientVpnRouteStatusNotFound = "NotFound"

	ClientVpnRouteStatusUnknown = "Unknown"
)

// ClientVpnRouteStatus fetches the Client VPN route and its Status
func ClientVpnRouteStatus(conn *ec2.EC2, routeID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		result, err := finder.ClientVpnRouteByID(conn, routeID)
		if tfawserr.ErrCodeEquals(err, tfec2.ErrCodeClientVpnRouteNotFound) {
			return nil, ClientVpnRouteStatusNotFound, nil
		}
		if err != nil {
			return nil, ClientVpnRouteStatusUnknown, err
		}

		if result == nil || len(result.Routes) == 0 || result.Routes[0] == nil {
			return nil, ClientVpnRouteStatusNotFound, nil
		}

		if len(result.Routes) > 1 {
			return nil, ClientVpnRouteStatusUnknown, fmt.Errorf("internal error: found %d results for Client VPN route (%s) status, need 1", len(result.Routes), routeID)
		}

		rule := result.Routes[0]
		if rule.Status == nil || rule.Status.Code == nil {
			return rule, ClientVpnRouteStatusUnknown, nil
		}

		return rule, aws.StringValue(rule.Status.Code), nil
	}
}

// InstanceIamInstanceProfile fetches the Instance and its IamInstanceProfile
//
// The EC2 API accepts a name and always returns an ARN, so it is converted
// back to the name to prevent unexpected differences.
func InstanceIamInstanceProfile(conn *ec2.EC2, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		instance, err := finder.InstanceByID(conn, id)

		if tfawserr.ErrCodeEquals(err, tfec2.ErrCodeInvalidInstanceIDNotFound) {
			return nil, "", nil
		}

		if err != nil {
			return nil, "", err
		}

		if instance == nil {
			return nil, "", nil
		}

		if instance.IamInstanceProfile == nil || instance.IamInstanceProfile.Arn == nil {
			return instance, "", nil
		}

		name, err := tfiam.InstanceProfileARNToName(aws.StringValue(instance.IamInstanceProfile.Arn))

		if err != nil {
			return instance, "", err
		}

		return instance, name, nil
	}
}

const (
	RouteStatusReady = "ready"
)

func RouteStatus(conn *ec2.EC2, routeFinder finder.RouteFinder, routeTableID, destination string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		output, err := routeFinder(conn, routeTableID, destination)

		if tfresource.NotFound(err) {
			return nil, "", nil
		}

		if err != nil {
			return nil, "", err
		}

		return output, RouteStatusReady, nil
	}
}

const (
	RouteTableStatusReady = "ready"
)

func RouteTableStatus(conn *ec2.EC2, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		output, err := finder.RouteTableByID(conn, id)

		if tfresource.NotFound(err) {
			return nil, "", nil
		}

		if err != nil {
			return nil, "", err
		}

		return output, RouteTableStatusReady, nil
	}
}

func RouteTableAssociationState(conn *ec2.EC2, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		output, err := finder.RouteTableAssociationByID(conn, id)

		if tfresource.NotFound(err) {
			return nil, "", nil
		}

		if err != nil {
			return nil, "", err
		}

		return output.AssociationState, aws.StringValue(output.AssociationState.State), nil
	}
}

const (
	SecurityGroupStatusCreated = "Created"

	SecurityGroupStatusNotFound = "NotFound"

	SecurityGroupStatusUnknown = "Unknown"
)

// SecurityGroupStatus fetches the security group and its status
func SecurityGroupStatus(conn *ec2.EC2, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		group, err := finder.SecurityGroupByID(conn, id)
		if tfresource.NotFound(err) {
			return nil, SecurityGroupStatusNotFound, nil
		}
		if err != nil {
			return nil, SecurityGroupStatusUnknown, err
		}

		return group, SecurityGroupStatusCreated, nil
	}
}

// SubnetMapCustomerOwnedIpOnLaunch fetches the Subnet and its MapCustomerOwnedIpOnLaunch
func SubnetMapCustomerOwnedIpOnLaunch(conn *ec2.EC2, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		subnet, err := finder.SubnetByID(conn, id)

		if tfawserr.ErrCodeEquals(err, tfec2.ErrCodeInvalidSubnetIDNotFound) {
			return nil, "false", nil
		}

		if err != nil {
			return nil, "false", err
		}

		if subnet == nil {
			return nil, "false", nil
		}

		return subnet, strconv.FormatBool(aws.BoolValue(subnet.MapCustomerOwnedIpOnLaunch)), nil
	}
}

// SubnetMapPublicIpOnLaunch fetches the Subnet and its MapPublicIpOnLaunch
func SubnetMapPublicIpOnLaunch(conn *ec2.EC2, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		subnet, err := finder.SubnetByID(conn, id)

		if tfawserr.ErrCodeEquals(err, tfec2.ErrCodeInvalidSubnetIDNotFound) {
			return nil, "false", nil
		}

		if err != nil {
			return nil, "false", err
		}

		if subnet == nil {
			return nil, "false", nil
		}

		return subnet, strconv.FormatBool(aws.BoolValue(subnet.MapPublicIpOnLaunch)), nil
	}
}

func TransitGatewayPrefixListReferenceState(conn *ec2.EC2, transitGatewayRouteTableID string, prefixListID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		transitGatewayPrefixListReference, err := finder.TransitGatewayPrefixListReference(conn, transitGatewayRouteTableID, prefixListID)

		if err != nil {
			return nil, "", err
		}

		if transitGatewayPrefixListReference == nil {
			return nil, "", nil
		}

		return transitGatewayPrefixListReference, aws.StringValue(transitGatewayPrefixListReference.State), nil
	}
}

func TransitGatewayRouteTablePropagationState(conn *ec2.EC2, transitGatewayRouteTableID string, transitGatewayAttachmentID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		transitGatewayRouteTablePropagation, err := finder.TransitGatewayRouteTablePropagation(conn, transitGatewayRouteTableID, transitGatewayAttachmentID)

		if err != nil {
			return nil, "", err
		}

		if transitGatewayRouteTablePropagation == nil {
			return nil, "", nil
		}

		return transitGatewayRouteTablePropagation, aws.StringValue(transitGatewayRouteTablePropagation.State), nil
	}
}

// VpcAttribute fetches the Vpc and its attribute value
func VpcAttribute(conn *ec2.EC2, id string, attribute string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		attributeValue, err := finder.VpcAttribute(conn, id, attribute)

		if tfawserr.ErrCodeEquals(err, tfec2.ErrCodeInvalidVpcIDNotFound) {
			return nil, "", nil
		}

		if err != nil {
			return nil, "", err
		}

		if attributeValue == nil {
			return nil, "", nil
		}

		return attributeValue, strconv.FormatBool(aws.BoolValue(attributeValue)), nil
	}
}

const (
	vpcPeeringConnectionStatusNotFound = "NotFound"
	vpcPeeringConnectionStatusUnknown  = "Unknown"
)

// VpcPeeringConnectionStatus fetches the VPC peering connection and its status
func VpcPeeringConnectionStatus(conn *ec2.EC2, vpcPeeringConnectionID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		vpcPeeringConnection, err := finder.VpcPeeringConnectionByID(conn, vpcPeeringConnectionID)
		if tfawserr.ErrCodeEquals(err, tfec2.ErrCodeInvalidVpcPeeringConnectionIDNotFound) {
			return nil, vpcPeeringConnectionStatusNotFound, nil
		}
		if err != nil {
			return nil, vpcPeeringConnectionStatusUnknown, err
		}

		// Sometimes AWS just has consistency issues and doesn't see
		// our peering connection yet. Return an empty state.
		if vpcPeeringConnection == nil || vpcPeeringConnection.Status == nil {
			return nil, vpcPeeringConnectionStatusNotFound, nil
		}

		statusCode := aws.StringValue(vpcPeeringConnection.Status.Code)

		// https://docs.aws.amazon.com/vpc/latest/peering/vpc-peering-basics.html#vpc-peering-lifecycle
		switch statusCode {
		case ec2.VpcPeeringConnectionStateReasonCodeFailed:
			log.Printf("[WARN] VPC Peering Connection (%s): %s: %s", vpcPeeringConnectionID, statusCode, aws.StringValue(vpcPeeringConnection.Status.Message))
			fallthrough
		case ec2.VpcPeeringConnectionStateReasonCodeDeleted, ec2.VpcPeeringConnectionStateReasonCodeExpired, ec2.VpcPeeringConnectionStateReasonCodeRejected:
			return nil, vpcPeeringConnectionStatusNotFound, nil
		}

		return vpcPeeringConnection, statusCode, nil
	}
}

const (
	attachmentStateNotFound = "NotFound"
	attachmentStateUnknown  = "Unknown"
)

// VpnGatewayVpcAttachmentState fetches the attachment between the specified VPN gateway and VPC and its state
func VpnGatewayVpcAttachmentState(conn *ec2.EC2, vpnGatewayID, vpcID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		vpcAttachment, err := finder.VpnGatewayVpcAttachment(conn, vpnGatewayID, vpcID)
		if tfawserr.ErrCodeEquals(err, tfec2.InvalidVpnGatewayIDNotFound) {
			return nil, attachmentStateNotFound, nil
		}
		if err != nil {
			return nil, attachmentStateUnknown, err
		}

		if vpcAttachment == nil {
			return nil, attachmentStateNotFound, nil
		}

		return vpcAttachment, aws.StringValue(vpcAttachment.State), nil
	}
}

func HostState(conn *ec2.EC2, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		output, err := finder.HostByID(conn, id)

		if tfresource.NotFound(err) {
			return nil, "", nil
		}

		if err != nil {
			return nil, "", err
		}

		return output, aws.StringValue(output.State), nil
	}
}

func ManagedPrefixListState(conn *ec2.EC2, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		output, err := finder.ManagedPrefixListByID(conn, id)

		if tfresource.NotFound(err) {
			return nil, "", nil
		}

		if err != nil {
			return nil, "", err
		}

		return output, aws.StringValue(output.State), nil
	}
}

func VpcEndpointState(conn *ec2.EC2, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		output, err := finder.VpcEndpointByID(conn, id)

		if tfresource.NotFound(err) {
			return nil, "", nil
		}

		if err != nil {
			return nil, "", err
		}

		return output, aws.StringValue(output.State), nil
	}
}

const (
	VpcEndpointRouteTableAssociationStatusReady = "ready"
)

func VpcEndpointRouteTableAssociationStatus(conn *ec2.EC2, vpcEndpointID, routeTableID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		err := finder.VpcEndpointRouteTableAssociationExists(conn, vpcEndpointID, routeTableID)

		if tfresource.NotFound(err) {
			return nil, "", nil
		}

		if err != nil {
			return nil, "", err
		}

		return "", VpcEndpointRouteTableAssociationStatusReady, nil
	}
}
