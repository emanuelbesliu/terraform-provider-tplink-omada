package resources

import (
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"

	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

// siteIDResourceSchema returns the schema attribute for site_id on resources.
// The site cannot change after creation (RequiresReplace).
func siteIDResourceSchema() schema.StringAttribute {
	return schema.StringAttribute{
		Description: "The ID of the site this resource belongs to.",
		Required:    true,
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.RequiresReplace(),
		},
	}
}

// siteIDDataSourceSchema returns the schema attribute for site_id on data sources.
func siteIDDataSourceSchema() dsschema.StringAttribute {
	return dsschema.StringAttribute{
		Description: "The ID of the site to query.",
		Required:    true,
	}
}

// parseImportID splits an import ID of the form "siteID/resourceID".
func parseImportID(importID string) (siteID, resourceID string, ok bool) {
	parts := strings.SplitN(importID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

// parseImportID3 splits an import ID of the form "siteID/part1/part2".
func parseImportID3(importID string) (siteID, part1, part2 string, ok bool) {
	parts := strings.SplitN(importID, "/", 3)
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return "", "", "", false
	}
	return parts[0], parts[1], parts[2], true
}
