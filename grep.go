package main

import (
	"bufio"
	"bytes"
	"container/list"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"

	yaml "gopkg.in/yaml.v3"

	"github.com/fatih/color"
	"github.com/mikefarah/yq/v4/pkg/yqlib"

	cli "github.com/urfave/cli/v2"
)

// Version is the version of the build.
const Version = "v0.1"

type ReleaseInfo struct {
	Name       string `json:"name"`
	Namespace  string `json:"namespace"`
	Revision   string `json:"revision"`
	Updated    string `json:"updated"`
	Status     string `json:"status"`
	AppVersion string `json:"app_version"`
	Chart      string `json:"chart"`
}

func yamlToProps(sampleYaml string) string {
	var output bytes.Buffer
	writer := bufio.NewWriter(&output)

	var propsEncoder = yqlib.NewPropertiesEncoder(writer)
	// inputs, err := yqlib.readDocuments(strings.NewReader(sampleYaml), "sample.yml", 0)
	inputs, err := MyreadDocuments(strings.NewReader(sampleYaml), "sample.yml", 0)
	if err != nil {
		panic(err)
	}
	node := inputs.Front().Value.(*yqlib.CandidateNode).Node
	err = propsEncoder.Encode(node)
	if err != nil {
		panic(err)
	}
	writer.Flush()

	return strings.TrimSuffix(output.String(), "\n")
}

func MyreadDocuments(reader io.Reader, filename string, fileIndex int) (*list.List, error) {
	decoder := yaml.NewDecoder(reader)
	inputList := list.New()
	var currentIndex uint = 0

	for {
		var dataBucket yaml.Node
		errorReading := decoder.Decode(&dataBucket)

		if errorReading == io.EOF {
			switch reader := reader.(type) {
			case *os.File:
				safelyCloseFile(reader)
			}
			return inputList, nil
		} else if errorReading != nil {
			return nil, errorReading
		}
		candidateNode := &yqlib.CandidateNode{
			Document:         currentIndex,
			Filename:         filename,
			Node:             &dataBucket,
			FileIndex:        fileIndex,
			EvaluateTogether: true,
		}

		inputList.PushBack(candidateNode)

		currentIndex = currentIndex + 1
	}
}

func safelyCloseFile(file *os.File) {
	err := file.Close()
	if err != nil {
		// log.Error("Error closing file!")
		// log.Error(err.Error())
		fmt.Println("errro for safelyCloseFile")
	}
}

func listRelease(namespace string) ([]ReleaseInfo, error) {
	args := []string{"list"}
	if namespace == "all" {
		args = append(args, "--all-namespaces", "-o", "json")
	} else {
		args = append(args, "--namespace", namespace, "-o", "json")
	}

	cmd := exec.Command(os.Getenv("HELM_BIN"), args...)
	//return outputWithRichError(cmd)
	release_arr, err := outputWithRichError(cmd)
	if err != nil {
		log.Fatal(err)
	}

	var releaseInfo []ReleaseInfo

	ok := json.Unmarshal(release_arr, &releaseInfo)
	if ok != nil {
		return nil, ok
	}
	return releaseInfo, nil
}

func getReleaseValues(release, namespace string) ([]byte, error) {
	args := []string{"get", "values", "--all", release}
	if namespace != "" {
		args = append(args, "--namespace", namespace)
	}
	cmd := exec.Command(os.Getenv("HELM_BIN"), args...)
	return outputWithRichError(cmd)
}

func isDebug() bool {
	return os.Getenv("HELM_DEBUG") == "true"
}
func debugPrint(format string, a ...interface{}) {
	if isDebug() {
		fmt.Printf(format+"\n", a...)
	}
}

func outputWithRichError(cmd *exec.Cmd) ([]byte, error) {
	debugPrint("Executing %s", strings.Join(cmd.Args, " "))
	output, err := cmd.Output()
	if exitError, ok := err.(*exec.ExitError); ok {
		return output, fmt.Errorf("%s: %s", exitError.Error(), string(exitError.Stderr))
	}
	return output, err
}

func grep(namespace, release, filter string) {
	if namespace == "" {
		namespace = "default"
	}

	if release == "" {
		lr, err := listRelease(namespace)
		if err != nil {
			log.Fatal(err)
		}
		for i := range lr {
			avyaml, err := getReleaseValues(lr[i].Name, lr[i].Namespace)
			if err != nil {
				log.Fatal(err)
			}
			printone(lr[i].Namespace, lr[i].Name, string(avyaml), filter)
		}
	} else {
		avyaml, err := getReleaseValues(release, namespace)
		if err != nil {
			log.Fatal(err)
		}
		printone(namespace, release, string(avyaml), filter)
	}

	//fmt.Println(yamlToProps(string(avyaml)))

}

func printone(namespace, release, avyaml, filterStr string) {

	avpropslines := strings.Split(yamlToProps(avyaml), "\n")
	var printStrs []string
	//逐行过滤是否包含关键字
	for _, avpropsline := range avpropslines {
		//如果包含关键字，使关键字高亮显示
		if strings.Contains(avpropsline, filterStr) {
			//高亮后的待打印字符串
			printStr := ""
			//打印行拆分成数组
			printArr := strings.Split(avpropsline, filterStr)
			for index, noStr := range printArr {
				//最后元素不用高亮
				if index == len(printArr)-1 {
					printStr += fmt.Sprint(noStr)
					break
				}
				printStr += fmt.Sprint(noStr)
				printStr += fmt.Sprint(color.RedString(filterStr))
			}
			//fmt.Println(printStr)
			printStrs = append(printStrs, printStr)
		}
	}

	if len(printStrs) != 0 {
		color.Green("*****************" + "  " + namespace + "." + release + "  " + "****************")
	}

	for _, x := range printStrs {
		fmt.Println(x)
	}

}

func main() {
	var namespace string
	var release string

	app := &cli.App{
		Name:      "grep",
		Usage:     "A helm plugin :  release values searcher",
		UsageText: "helm grep -n {namespace} -r {release} {key0} {key1}",
		ArgsUsage: "helm grep -n {namespace} -r {release} {key0} {key1}",
		Authors: []*cli.Author{
			&cli.Author{
				Name:  "zhenghanchao",
				Email: "zhenghanchao@baidu.com",
			},
		},
		HideHelpCommand: true,
		Version:         Version,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "grepnamespace",
				Aliases:     []string{"ns", "grepns", "gns"},
				Usage:       "kubernetes namespace, default is default",
				Destination: &namespace,
			},
			&cli.StringFlag{
				Name:        "release",
				Aliases:     []string{"r"},
				Usage:       "helm release, Default is all release in namespace that u define",
				Destination: &release,
			},
		},

		Action: func(c *cli.Context) error {
			// hgrep(c.String("namespace"), c.String("release"), c.Args().First())
			// Args 是出你设定的参数外，多余输入的东西
			if c.Args().Len() == 0 {
				cmdNotRight(c)
			}
			//namespace和默认的helm参数冲突
			//grep(os.Getenv("HELM_NAMESPACE"), c.String("release"), c.Args().First())
			grep(c.String("ns"), c.String("release"), c.Args().First())
			return nil
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func cmdNotRight(c *cli.Context) {
	fmt.Printf(
		"your command is missing input. See '%s --help'.\n",
		c.App.Name,
	)
	os.Exit(1)
}
