package acp

import (
	"context"

	acpsdk "github.com/coder/acp-go-sdk"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/ports"
)

// ListSkills exposes ACP's available commands through Kennel's existing named-skill
// boundary. Invocation needs no protocol-specific method: ACP commands are sent
// as ordinary prompt text beginning with /name, which is already how Kennel's
// composer inserts a skill.
func (c *conversation) ListSkills(ctx context.Context) ([]ports.ChatSkill, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return cloneSkills(c.skills), nil
}

func (c *conversation) replaceAvailableCommands(commands []acpsdk.AvailableCommand) {
	skills := make([]ports.ChatSkill, 0, len(commands))
	for _, command := range commands {
		if command.Name == "" {
			continue
		}
		inputHint := ""
		if command.Input != nil && command.Input.Unstructured != nil {
			inputHint = command.Input.Unstructured.Hint
		}
		skills = append(skills, ports.ChatSkill{
			Name:        command.Name,
			DisplayName: command.Name,
			Description: command.Description,
			InputHint:   inputHint,
			// ACP does not report command provenance. "agent" tells the user this
			// came from the live provider without leaking protocol jargon into UI.
			Source: "agent",
		})
	}

	c.mu.Lock()
	c.skills = skills
	c.skillsKnown = true
	if c.capabilities != nil {
		c.capabilities[ports.ChatCapabilitySkills] = true
	}
	c.mu.Unlock()
}

func cloneSkills(skills []ports.ChatSkill) []ports.ChatSkill {
	return append([]ports.ChatSkill(nil), skills...)
}
