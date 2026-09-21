package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/KimHG1995/agent-bench/internal/domain"
	"github.com/KimHG1995/agent-bench/internal/openaiadapter"
)

func main() {
	var req domain.RunRequest
	if err := json.NewDecoder(os.Stdin).Decode(&req); err != nil {
		fmt.Fprintln(os.Stderr, "decode request:", err)
		os.Exit(1)
	}

	out, err := openaiadapter.Run(context.Background(), req, openaiadapter.ConfigFromEnv())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := json.NewEncoder(os.Stdout).Encode(out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
