package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string
var Debug bool

// rootCmd 代表基础命令，在不带任何子命令的情况下调用
var rootCmd = &cobra.Command{
	Use:   "geektime-go",
	Short: "极客时间课程下载工具",
	Long: `geektime-go 是一个命令行工具，可以帮助您获取
已购买的极客时间课程，并将其转换为本地 Markdown 文件。`,
	// 重写帮助模板以确保更多内容显示为中文
}

// Execute 将所有子命令添加到根命令中并适当设置标志。
// 这由 main.main() 调用。它只需对 rootCmd 执行一次。
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// 定义持久性标志
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "配置文件路径 (默认是 $HOME/.geektime-go.yaml)")
	rootCmd.PersistentFlags().BoolVar(&Debug, "debug", false, "开启调试模式，显示详细响应数据")

	// 根命令的本地标志
	rootCmd.Flags().BoolP("toggle", "t", false, "显示帮助信息")

	// 设置自定义的用法提示（部分英文是 Cobra 内部硬编码的，但我们可以通过设置自定义模板来优化）
	rootCmd.SetHelpCommand(&cobra.Command{
		Use:    "help [命令]",
		Short:  "显示命令的帮助信息",
		Long:   `显示任何命令的详细说明。`,
		Run: func(c *cobra.Command, args []string) {
			cmd, _, _ := c.Root().Find(args)
			if cmd == nil {
				c.Printf("未找到命令 %q\n", args[0])
				return
			}
			cmd.HelpFunc()(cmd, args)
		},
	})
}

// initConfig 读取配置文件和环境变量（如果已设置）
func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".geektime-go")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		if Debug {
			fmt.Fprintln(os.Stderr, "使用配置文件:", viper.ConfigFileUsed())
		}
	}
}
