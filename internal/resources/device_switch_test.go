package resources

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func TestSwitchPortSpeedReadOnly(t *testing.T) {
	// Omada's speed is the observed link speed, not the configured linkSpeed.
	// Allowing it in configuration turns link negotiation into persistent drift.
	speed := switchPortSchema.Attributes["speed"].(schema.Int64Attribute)
	if !speed.Computed || speed.Optional || speed.Required {
		t.Fatal("operational port speed must be computed-only")
	}
}
