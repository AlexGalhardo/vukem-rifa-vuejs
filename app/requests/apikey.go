package requests

import "github.com/AlexGalhardo/vukem/z"

// API key abilities (PLAN.md §5 REST table).
const (
	AbilityRafflesRead  = "raffles:read"
	AbilityRafflesWrite = "raffles:write"
)

// Abilities lists every ability an API key may hold.
func Abilities() []string { return []string{AbilityRafflesRead, AbilityRafflesWrite} }

// APIKey is the "new API key" form.
type APIKey struct {
	Name      string   `json:"name"      form:"name"`
	Abilities []string `json:"abilities" form:"abilities"`
}

// APIKeySchema validates APIKey: a name and at least one known ability.
var APIKeySchema = z.Struct[APIKey](z.Shape{
	"name":      z.String().Trim().Min(1).Max(60),
	"abilities": z.Slice(z.Enum(AbilityRafflesRead, AbilityRafflesWrite)).Min(1).Max(2).Unique(),
})
