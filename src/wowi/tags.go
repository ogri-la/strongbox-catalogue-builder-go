package wowi

// WoWInterface-specific category replacement map
// Categories that are replaced entirely with specific tags
var wowiReplacements = map[string][]string{
	"Character Advancement":   {"quests", "leveling", "achievements"},
	"Other":                   {"misc"},
	"Suites":                  {"compilations"},
	"Graphic UI Mods":         {"ui", "ui-replacements"},
	"UI Media":                {"ui"},
	"ROFL":                    {"misc", "mini-games"},
	"Combat Mods":             {"combat"},
	"Buff, Debuff, Spell":     {"buffs", "debuffs"},
	"Casting Bars, Cooldowns": {"buffs", "debuffs", "ui"},
	"Map, Coords, Compasses":  {"map", "minimap", "coords", "ui"},
	"RolePlay, Music Mods":    {"role-play", "audio"},
	"Chat Mods":               {"chat"},
	"Unit Mods":               {"unit-frames"},
	"Raid Mods":               {"unit-frames", "raid-frames"},
	"Data Mods":               {"data"},
	"Utility Mods":            {"utility"},
	"Action Bar Mods":         {"action-bars", "ui"},
	"Tradeskill Mods":         {"tradeskill"},
	"Classic - General":       {"classic"},
}

// WoWInterface-specific category supplement map
// Categories that gain additional tags (don't replace the category itself)
var wowiSupplements = map[string][]string{
	"Pets":        {"battle-pets", "companions"},
	"Data Broker": {"data"},
	"Titan Panel": {"plugins"},
	"FuBar":       {"plugins"},
	"Mail":        {"ui"},
}

// categoryToTagsWithMaps converts a WowInterface category to tags using replacement/supplement maps
// Following the Clojure implementation:
// 1. Check if category has a replacement mapping - if so, use those tags
// 2. Check if category has supplementary tags - add those
// 3. If no replacement found, split category on " & ", ", ", ": " and convert each part
func categoryToTagsWithMaps(category string) []string {
	// Check for replacement tags
	if replacementTags, hasReplacement := wowiReplacements[category]; hasReplacement {
		// Check for supplementary tags to add
		if supplementaryTags, hasSupplement := wowiSupplements[category]; hasSupplement {
			// Combine replacement and supplement tags
			allTags := make([]string, 0, len(replacementTags)+len(supplementaryTags))
			allTags = append(allTags, replacementTags...)
			allTags = append(allTags, supplementaryTags...)
			return allTags
		}
		return replacementTags
	}

	// No replacement, check for supplements only
	var tagList []string
	if supplementaryTags, hasSupplement := wowiSupplements[category]; hasSupplement {
		tagList = append(tagList, supplementaryTags...)
	}

	// Split the category and convert each part to a tag
	splitTags := categoryToTags(category)
	tagList = append(tagList, splitTags...)

	return tagList
}
