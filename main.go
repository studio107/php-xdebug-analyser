package main

import (
	"github.com/cheggaaa/pb"
	"github.com/codegangsta/cli"
	"github.com/crackcomm/go-clitable"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
)

var (
	functions      = make(map[string]*Result)
	stackFunctions = make(map[string]string)
)

// function #calls time memory time memory

func ParseFile(path string) ([]string, error) {
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		return []string{}, err
	}
	return strings.Split(string(contents), "\n"), nil
}

type Result struct {
	FuncName        string
	Calls           int
	TimeInclusive   float64
	MemoryInclusive int
	TimeChildren    float64
	MemoryChildren  int
	TimeOwn         float64
	MemoryOwn       int
}

type ResultSlice []*Result

// Len is part of sort.Interface.
func (r ResultSlice) Len() int {
	return len(r)
}

// Less is part of sort.Interface. We use count as the value to sort by
func (r ResultSlice) Less(i, j int) bool {
	return r[i].Calls < r[j].Calls
}

// Swap is part of sort.Interface.
func (r ResultSlice) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

func addToFunction(function string, time float64, memory int, nestedTime float64, nestedMemory int) {
	var result *Result
	var ok bool
	result, ok = functions[function]
	if !ok {
		result = &Result{
			FuncName:        function,
			Calls:           0,
			TimeInclusive:   0.0,
			MemoryInclusive: 0,
			TimeChildren:    0.0,
			MemoryChildren:  0,
			TimeOwn:         0.0,
			MemoryOwn:       0,
		}
		functions[function] = result
	}

	result.Calls += 1
	_, has := stackFunctions[function]
	if !has {
		result.TimeInclusive += time
		result.MemoryInclusive += memory
		result.TimeChildren += nestedTime
		result.MemoryChildren += nestedMemory
	}
}

func main() {
	app := cli.NewApp()
	app.Name = "php xdebug analyzer"
	app.Usage = "php is sucks"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "file",
			Value: "",
			Usage: "path to file",
		},
	}
	app.Action = func(c *cli.Context) {
		var stack = make(map[int][]string)

		stack[-1] = []string{"", "0", "0", "0", "0"}
		stack[0] = []string{"", "0", "0", "0", "0"}

		path := c.String("file")
		if path == "" {
			log.Printf("Please set file")
			os.Exit(2)
		}

		lines, err := ParseFile(path)
		if err != nil {
			panic(err)
		}

		log.Printf("Start")

		count := len(lines)
		bar := pb.StartNew(count)
		for _, line := range lines {
			bar.Increment()

			parts := strings.Split(line, "\t")
			if len(parts) < 5 {
				continue
			}

			// TODO refact

			depthStr := parts[0]
			depth, _ := strconv.Atoi(depthStr)
			timeStr := parts[3]
			time, _ := strconv.ParseFloat(timeStr, 64)
			memoryStr := parts[4]
			memory, _ := strconv.Atoi(memoryStr)

			if parts[2] == "0" {
				funcName := parts[5]
				stack[depth] = []string{funcName, timeStr, memoryStr, "0", "0"}
				stackFunctions[funcName] = funcName
			} else if parts[2] == "1" {
				funcName := stack[depth][0]
				prevTimeStr := stack[depth][1]
				prevTime, _ := strconv.ParseFloat(prevTimeStr, 64)
				prevMemStr := stack[depth][2]
				prevMem, _ := strconv.Atoi(prevMemStr)
				nestedTimeStr := stack[depth][3]
				nestedTime, _ := strconv.ParseFloat(nestedTimeStr, 64)
				nestedMemoryStr := stack[depth][4]
				nestedMemory, _ := strconv.Atoi(nestedMemoryStr)

				dTime := time - prevTime
				dMemory := memory - prevMem

				// stack[depth-1][3] += dTime
				// stack[depth-1][4] += dMemory

				delete(stackFunctions, funcName)
				addToFunction(funcName, dTime, dMemory, nestedTime, nestedMemory)
			}
		}
		bar.FinishPrint("Complete.")

		var rows []map[string]interface{}

		s := make(ResultSlice, 0, len(functions))
		for _, f := range functions {
			s = append(s, f)
		}

		sort.Sort(s)
		for i, v := range s {
			var row = make(map[string]interface{})
			if v.Calls == 1 {
				continue
			}
			row["name"] = v.FuncName
			row["calls"] = strconv.Itoa(v.Calls)
			row["time"] = strconv.FormatFloat(v.TimeInclusive, 'f', 6, 64)
			row["memory"] = strconv.Itoa(v.MemoryInclusive)
			row["timeChild"] = strconv.FormatFloat(v.TimeChildren, 'f', 6, 64)
			row["memoryChild"] = strconv.Itoa(v.MemoryChildren)
			rows = append(rows, row)
		}
		log.Printf("%d", len(rows))
		clitable.PrintTable([]string{"name", "calls", "time", "memory", "timeChild", "memoryChild"}, rows)
	}

	app.Run(os.Args)
}
