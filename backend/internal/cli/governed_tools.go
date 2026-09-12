package cli

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/Pin4sf/Waldo-Kennel/backend/internal/domain"
	"github.com/Pin4sf/Waldo-Kennel/backend/internal/governedtools"
)

func newGovernedToolsCommand(ctx *commandContext) *cobra.Command {
	var workspace, encodedPolicy string
	cmd := &cobra.Command{Use: "governed-tools", Hidden: true, Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		raw, err := base64.RawURLEncoding.DecodeString(encodedPolicy)
		if err != nil {
			return fmt.Errorf("decode governed policy: %w", err)
		}
		var policy domain.AttemptExecutionPolicy
		if err := json.Unmarshal(raw, &policy); err != nil {
			return fmt.Errorf("decode governed policy: %w", err)
		}
		return (governedtools.Server{Policy: policy, WorkspaceRoot: workspace, In: ctx.deps.In, Out: ctx.deps.Out}).Serve(cmd.Context())
	}}
	cmd.Flags().StringVar(&workspace, "workspace", "", "leased workspace root")
	cmd.Flags().StringVar(&encodedPolicy, "policy", "", "base64url frozen execution policy")
	_ = cmd.MarkFlagRequired("workspace")
	_ = cmd.MarkFlagRequired("policy")
	return cmd
}
