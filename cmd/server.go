/*
Copyright © 2024 raul <raul@bulgariu.xyz
*/

package cmd

import (
	"github.com/spf13/cobra"
	"log"
)

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Tiamat Server",
	Long:  `Tiamat Server including the C2 server and the Web server`,
	Run: func(cmd *cobra.Command, args []string) {

		if err := setServerParameters(cmd); err != nil {
			log.Fatalf("Error happened trying to set parameters: %v\n", err)
		}

		Server()
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)
	serverCmd.PersistentFlags().StringP("clients", "c", "", "File to recover clients from")
	serverCmd.Flags().Bool("insecure", false, "[UNSAFE] Do not use TLS encryption")
}

func setServerParameters(cmd *cobra.Command) error {
	parameterClients, err := cmd.Flags().GetString("clients")
	if err != nil {
		return err
	}
	if parameterClients != "" {
		isUsingJSONParameter = true
		clientJSONPath = parameterClients
	}
	insecure, err := cmd.Flags().GetBool("insecure")
	if insecure == true {
		servInsecure = true
	}
	return nil
}
