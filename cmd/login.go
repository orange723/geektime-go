package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// loginCmd 代表登录命令
var loginCmd = &cobra.Command{
	Use:   "login [Cookie内容]",
	Short: "配置您的极客时间 Cookie",
	Long:  `将您的极客时间 Cookie 保存到本地配置文件中，以便后续请求。`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cookie := strings.TrimSpace(args[0])
		viper.Set("cookie", cookie)

		home, err := os.UserHomeDir()
		cobra.CheckErr(err)
		configPath := home + "/.geektime-go.yaml"

		err = viper.WriteConfigAs(configPath)
		if err != nil {
			fmt.Println("保存配置出错:", err)
			return
		}
		fmt.Printf("Cookie 已成功保存到 %s\n", configPath)
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
}
