package nat_gateway

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/megaport/megaport-cli/internal/base/exitcodes"
	"github.com/megaport/megaport-cli/internal/utils"
	"github.com/megaport/megaport-cli/internal/validation"
	megaport "github.com/megaport/megaportgo"
	"github.com/spf13/cobra"
)

func processJSONCreateNATGatewayInput(jsonStr, jsonFile string) (*megaport.CreateNATGatewayRequest, error) {
	jsonData, err := utils.ReadJSONInput(jsonStr, jsonFile)
	if err != nil {
		return nil, exitcodes.NewUsageError(err)
	}

	// Unmarshal into a flat helper to handle camelCase JSON keys.
	var raw struct {
		Name                  string            `json:"name"`
		LocationID            int               `json:"locationId"`
		Speed                 int               `json:"speed"`
		Term                  int               `json:"term"`
		SessionCount          int               `json:"sessionCount"`
		DiversityZone         string            `json:"diversityZone"`
		ASN                   int               `json:"asn"`
		BGPShutdownDefault    bool              `json:"bgpShutdownDefault"`
		AutoRenewTerm         bool              `json:"autoRenewTerm"`
		PromoCode             string            `json:"promoCode"`
		ServiceLevelReference string            `json:"serviceLevelReference"`
		ResourceTags          map[string]string `json:"resourceTags"`
	}
	if err := json.Unmarshal(jsonData, &raw); err != nil {
		return nil, exitcodes.NewUsageError(fmt.Errorf("failed to parse JSON: %w", err))
	}

	if err := utils.RejectEmptyTagKeys(raw.ResourceTags); err != nil {
		return nil, exitcodes.NewUsageError(err)
	}

	req := &megaport.CreateNATGatewayRequest{
		ProductName:           raw.Name,
		LocationID:            raw.LocationID,
		Speed:                 raw.Speed,
		Term:                  raw.Term,
		AutoRenewTerm:         raw.AutoRenewTerm,
		PromoCode:             raw.PromoCode,
		ServiceLevelReference: raw.ServiceLevelReference,
		Config: megaport.NATGatewayNetworkConfig{
			ASN:                raw.ASN,
			BGPShutdownDefault: raw.BGPShutdownDefault,
			DiversityZone:      raw.DiversityZone,
			SessionCount:       raw.SessionCount,
		},
	}

	if raw.ResourceTags != nil {
		tags := make([]megaport.ResourceTag, 0, len(raw.ResourceTags))
		for k, v := range raw.ResourceTags {
			tags = append(tags, megaport.ResourceTag{Key: k, Value: v})
		}
		req.ResourceTags = tags
	}

	if err := validation.ValidateCreateNATGatewayRequest(req); err != nil {
		return nil, err
	}
	return req, nil
}

func processFlagCreateNATGatewayInput(cmd *cobra.Command) (*megaport.CreateNATGatewayRequest, error) {
	name, _ := cmd.Flags().GetString("name")
	term, _ := cmd.Flags().GetInt("term")
	speed, _ := cmd.Flags().GetInt("speed")
	locationID, _ := cmd.Flags().GetInt("location-id")
	sessionCount, _ := cmd.Flags().GetInt("session-count")
	asn, _ := cmd.Flags().GetInt("asn")
	diversityZone, _ := cmd.Flags().GetString("diversity-zone")
	promoCode, _ := cmd.Flags().GetString("promo-code")
	serviceLevelRef, _ := cmd.Flags().GetString("service-level-reference")
	autoRenew, _ := cmd.Flags().GetBool("auto-renew")

	resourceTagsStr, _ := cmd.Flags().GetString("resource-tags")
	resourceTagsFile, _ := cmd.Flags().GetString("resource-tags-file")
	var resourceTags []megaport.ResourceTag
	if resourceTagsStr != "" || resourceTagsFile != "" {
		tagData, err := utils.ReadJSONInput(resourceTagsStr, resourceTagsFile)
		if err != nil {
			return nil, exitcodes.NewUsageError(fmt.Errorf("failed to read resource tags: %w", err))
		}
		var tagsMap map[string]string
		if err := json.Unmarshal(tagData, &tagsMap); err != nil {
			return nil, exitcodes.NewUsageError(fmt.Errorf("failed to parse resource tags JSON: %w", err))
		}
		if err := utils.RejectEmptyTagKeys(tagsMap); err != nil {
			return nil, exitcodes.NewUsageError(err)
		}
		for k, v := range tagsMap {
			resourceTags = append(resourceTags, megaport.ResourceTag{Key: k, Value: v})
		}
	}

	req := &megaport.CreateNATGatewayRequest{
		ProductName:           name,
		LocationID:            locationID,
		Speed:                 speed,
		Term:                  term,
		AutoRenewTerm:         autoRenew,
		PromoCode:             promoCode,
		ServiceLevelReference: serviceLevelRef,
		ResourceTags:          resourceTags,
		Config: megaport.NATGatewayNetworkConfig{
			ASN:           asn,
			DiversityZone: diversityZone,
			SessionCount:  sessionCount,
		},
	}

	if err := validation.ValidateCreateNATGatewayRequest(req); err != nil {
		return nil, err
	}
	return req, nil
}

// updateExplicitFields tracks which zero-valued fields were explicitly provided
// in an update request. This lets mergeUpdateDefaults distinguish "user wants
// zero/false/empty" from "user omitted the field and we should inherit from
// the original resource".
//
// Fields that use pointer types in the raw JSON struct (ASN *int,
// SessionCount *int, DiversityZone *string, AutoRenewTerm *bool,
// BGPShutdownDefault *bool) get a corresponding entry here because their zero
// value is ambiguous. Fields with genuinely invalid zero values
// (LocationID, Speed, Term — all must be positive) do not need tracking
// because 0 can only mean "omitted". PromoCode and ServiceLevelReference are
// likewise plain strings; mergeUpdateDefaults inherits them from the original
// resource when left blank rather than relying on server-side omitempty
// behavior.
type updateExplicitFields struct {
	AutoRenewTerm         bool // was autoRenewTerm present in input?
	BGPShutdownDefault    bool // was bgpShutdownDefault present in input?
	ASN                   bool // was asn present in JSON input? (no flag path for ASN)
	SessionCount          bool // was sessionCount present in JSON input?
	DiversityZone         bool // was diversityZone present in JSON input?
	PromoCode             bool // was promoCode present in input?
	ServiceLevelReference bool // was serviceLevelReference present in input?
}

// processJSONUpdateNATGatewayInput parses a JSON update request and returns the
// request along with explicit-presence flags for ambiguous fields. Fields whose
// zero value could be a valid intentional value (bool, int for ASN/SessionCount,
// string for DiversityZone) use pointer types in the raw struct so nil means
// "not provided" and mergeUpdateDefaults will inherit from the original resource.
// Fields with genuinely invalid zero values (LocationID, Speed, Term) use plain
// types. PromoCode and ServiceLevelReference use pointer types too: they carry
// omitempty in the SDK request, so an explicit "" is dropped from the wire and
// the server clears the field, but only if we don't inherit the original first.
func processJSONUpdateNATGatewayInput(jsonStr, jsonFile, uid string) (*megaport.UpdateNATGatewayRequest, updateExplicitFields, error) {
	jsonData, err := utils.ReadJSONInput(jsonStr, jsonFile)
	if err != nil {
		return nil, updateExplicitFields{}, err
	}

	var raw struct {
		Name                  string            `json:"name"`
		LocationID            int               `json:"locationId"`
		Speed                 int               `json:"speed"`
		Term                  int               `json:"term"`
		SessionCount          *int              `json:"sessionCount"`
		DiversityZone         *string           `json:"diversityZone"`
		ASN                   *int              `json:"asn"`
		BGPShutdownDefault    *bool             `json:"bgpShutdownDefault"`
		AutoRenewTerm         *bool             `json:"autoRenewTerm"`
		PromoCode             *string           `json:"promoCode"`
		ServiceLevelReference *string           `json:"serviceLevelReference"`
		ResourceTags          map[string]string `json:"resourceTags"`
	}
	if err := json.Unmarshal(jsonData, &raw); err != nil {
		return nil, updateExplicitFields{}, fmt.Errorf("failed to parse JSON: %w", err)
	}

	if err := utils.RejectEmptyTagKeys(raw.ResourceTags); err != nil {
		return nil, updateExplicitFields{}, exitcodes.NewUsageError(err)
	}

	explicit := updateExplicitFields{
		AutoRenewTerm:         raw.AutoRenewTerm != nil,
		BGPShutdownDefault:    raw.BGPShutdownDefault != nil,
		ASN:                   raw.ASN != nil,
		SessionCount:          raw.SessionCount != nil,
		DiversityZone:         raw.DiversityZone != nil,
		PromoCode:             raw.PromoCode != nil,
		ServiceLevelReference: raw.ServiceLevelReference != nil,
	}

	var autoRenew, bgpShutdown bool
	if raw.AutoRenewTerm != nil {
		autoRenew = *raw.AutoRenewTerm
	}
	if raw.BGPShutdownDefault != nil {
		bgpShutdown = *raw.BGPShutdownDefault
	}
	var asn, sessionCount int
	if raw.ASN != nil {
		asn = *raw.ASN
	}
	if raw.SessionCount != nil {
		sessionCount = *raw.SessionCount
	}
	var diversityZone string
	if raw.DiversityZone != nil {
		diversityZone = *raw.DiversityZone
	}
	var promoCode, serviceLevelRef string
	if raw.PromoCode != nil {
		promoCode = *raw.PromoCode
	}
	if raw.ServiceLevelReference != nil {
		serviceLevelRef = *raw.ServiceLevelReference
	}

	req := &megaport.UpdateNATGatewayRequest{
		ProductUID:            uid,
		ProductName:           raw.Name,
		LocationID:            raw.LocationID,
		Speed:                 raw.Speed,
		Term:                  raw.Term,
		AutoRenewTerm:         autoRenew,
		PromoCode:             promoCode,
		ServiceLevelReference: serviceLevelRef,
		Config: megaport.NATGatewayNetworkConfig{
			ASN:                asn,
			BGPShutdownDefault: bgpShutdown,
			DiversityZone:      diversityZone,
			SessionCount:       sessionCount,
		},
	}

	if raw.ResourceTags != nil {
		tags := make([]megaport.ResourceTag, 0, len(raw.ResourceTags))
		for k, v := range raw.ResourceTags {
			tags = append(tags, megaport.ResourceTag{Key: k, Value: v})
		}
		req.ResourceTags = tags
	}

	return req, explicit, nil
}

func processFlagUpdateNATGatewayInput(cmd *cobra.Command, uid string) (*megaport.UpdateNATGatewayRequest, error) {
	if uid == "" {
		return nil, fmt.Errorf("product UID is required")
	}
	name, _ := cmd.Flags().GetString("name")
	term, _ := cmd.Flags().GetInt("term")
	speed, _ := cmd.Flags().GetInt("speed")
	locationID, _ := cmd.Flags().GetInt("location-id")
	sessionCount, _ := cmd.Flags().GetInt("session-count")
	diversityZone, _ := cmd.Flags().GetString("diversity-zone")
	promoCode, _ := cmd.Flags().GetString("promo-code")
	serviceLevelRef, _ := cmd.Flags().GetString("service-level-reference")
	autoRenew, _ := cmd.Flags().GetBool("auto-renew")

	resourceTagsStr, _ := cmd.Flags().GetString("resource-tags")
	resourceTagsFile, _ := cmd.Flags().GetString("resource-tags-file")
	var resourceTags []megaport.ResourceTag
	if resourceTagsStr != "" || resourceTagsFile != "" {
		tagData, err := utils.ReadJSONInput(resourceTagsStr, resourceTagsFile)
		if err != nil {
			return nil, exitcodes.NewUsageError(fmt.Errorf("failed to read resource tags: %w", err))
		}
		var tagsMap map[string]string
		if err := json.Unmarshal(tagData, &tagsMap); err != nil {
			return nil, exitcodes.NewUsageError(fmt.Errorf("failed to parse resource tags JSON: %w", err))
		}
		if err := utils.RejectEmptyTagKeys(tagsMap); err != nil {
			return nil, exitcodes.NewUsageError(err)
		}
		for k, v := range tagsMap {
			resourceTags = append(resourceTags, megaport.ResourceTag{Key: k, Value: v})
		}
	}

	req := &megaport.UpdateNATGatewayRequest{
		ProductUID:            uid,
		ProductName:           name,
		LocationID:            locationID,
		Speed:                 speed,
		Term:                  term,
		AutoRenewTerm:         autoRenew,
		PromoCode:             promoCode,
		ServiceLevelReference: serviceLevelRef,
		ResourceTags:          resourceTags,
		Config: megaport.NATGatewayNetworkConfig{
			DiversityZone: diversityZone,
			SessionCount:  sessionCount,
		},
	}

	return req, nil
}

// parseTelemetryTypes splits a comma-separated types string into a slice.
func parseTelemetryTypes(typesStr string) []string {
	if typesStr == "" {
		return nil
	}
	parts := strings.Split(typesStr, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(strings.ToUpper(p))
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
