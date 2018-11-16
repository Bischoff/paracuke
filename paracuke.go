// paracuke, still very experimental

package paracuke

import (
  "os"
  "fmt"
  "bufio"
  "bytes"
  "io"
  "strings"
  "regexp"
  "sync"
  "time"
)

type StepFunction func(context string, args []string) bool

type RegisteredStep struct {
  re *regexp.Regexp
  step StepFunction
}

type Context struct {
  name string
  features []string
}

type Scenario struct {
  name string
  steps []string
}

var registeredSteps []RegisteredStep = []RegisteredStep{}

var wg sync.WaitGroup

// Cucumber emulation
func registerStep(reStr string, stepFunc StepFunction) {
  registeredSteps = append(registeredSteps, RegisteredStep { re: regexp.MustCompile(reStr), step: stepFunc })
}

func Given(reStr string, stepFunc StepFunction) {
  registerStep(reStr, stepFunc)
}

func When(reStr string, stepFunc StepFunction) {
  registerStep(reStr, stepFunc)
}

func Then(reStr string, stepFunc StepFunction) {
  registerStep(reStr, stepFunc)
}

// SyntaxError
func syntaxError() {
  fmt.Fprintf(os.Stderr, "Syntax: %s [-v|-d] [<contexts>]\n", os.Args[0])
  fmt.Fprintf(os.Stderr, "  -v: show version\n")
  fmt.Fprintf(os.Stderr, "  -d: debug mode\n")
  fmt.Fprintf(os.Stderr, "  contexts: test contexts file\n")
  os.Exit(1)
}

// Error reading the contexts
func contextsReadError(filename string, err error) {
  fmt.Fprintf(os.Stderr, "Unable to read contexts file \"%s\":\n", filename)
  fmt.Fprintf(os.Stderr, "  %s\n", err.Error())
  os.Exit(2)
}

// Syntax error in the contexts
func contextsSyntaxError(filename string, line string, linenum int) {
  fmt.Fprintf(os.Stderr, "Syntax error on line %d of contexts file \"%s\":\n", linenum, filename)
  fmt.Fprintf(os.Stderr, "  \"%s\"\n", line)
  os.Exit(2)
}

// Duplicate context name error
func duplicateContextError(filename string, line string, linenum int) {
  fmt.Fprintf(os.Stderr, "Duplicate context name on line %d of contexts file \"%s\":\n", linenum, filename)
  fmt.Fprintf(os.Stderr, "  \"%s\"\n", line)
  os.Exit(3)
}

// Error reading the scenarios
func scenariosReadError(filename string, err error) {
  fmt.Fprintf(os.Stderr, "Unable to read feature file \"%s\":\n", filename)
  fmt.Fprintf(os.Stderr, "  %s\n", err.Error())
  os.Exit(4)
}

// Syntax error in the scenarios
func scenariosSyntaxError(filename string, line string, linenum int) {
  fmt.Fprintf(os.Stderr, "Syntax error on line %d of feature file \"%s\":\n", linenum, filename)
  fmt.Fprintf(os.Stderr, "  \"%s\"\n", line)
  os.Exit(5)
}

// Check duplicate context name
func checkDuplicateContext(contexts *[]Context, name string, filename string, line string, linenum int) {
  for _, context := range *contexts {
    if (context.name == name) {
      duplicateContextError(filename, line, linenum)
    }
  }
}

// Execute a step
func executeStep(context string, stepTitle string, step StepFunction, args []string) bool {
  if step(context, args) {
    fmt.Printf("\x1b[32m(%s)   %s\x1b[30m\n", context, stepTitle)
    return true
  }
  fmt.Printf("\x1b[31m(%s)   %s\x1b[30m\n", context, stepTitle)
  fmt.Printf("\x1b[31m(%s)   Step failed!\x1b[30m\n", context)
  return false
}

// Start a feature
func startFeature(context string, featureTitle string) {
  fmt.Printf("\x1b[32m(%s)  %s\x1b[30m\n", context, featureTitle)
  dashes := strings.Repeat("-", len(featureTitle))
  fmt.Printf("\x1b[32m(%s)  %s\x1b[30m\n", context, dashes)
  fmt.Printf("\x1b[32m(%s)\x1b[30m\n", context)
}

// Start a scenario
func startScenario(context string, scenarioTitle string) {
  fmt.Printf("\x1b[32m(%s)  %s\x1b[30m\n", context, scenarioTitle)
}

// Skip a step
func skipStep(context string, stepTitle string) {
  fmt.Printf("\x1b[36m(%s)   %s\x1b[30m\n", context, stepTitle)
  fmt.Printf("\x1b[36m(%s)     (skipped...)\x1b[30m\n", context)
}

// Start a step
func startStep(context string, stepTitle string) bool {
  for _, reg := range registeredSteps {
    if reg.re.MatchString(stepTitle) {
      args := reg.re.FindStringSubmatch(stepTitle)
      return executeStep(context, stepTitle, reg.step, args)
    }
  }
  fmt.Printf("\n\x1b[31m(%s) *** Please implement step:\n   \"%s\"\x1b[30m\n\n", context, stepTitle)
  return false
}

// Debug details of a series of contexts
func debugContexts(title string, contexts *[]Context) {
  fmt.Printf("(debug) *** %s\n", title)
  fmt.Printf("(debug)\n")
  for _, context := range *contexts {
    fmt.Printf("(debug) context \"%s\":\n", context.name)
    for _, feature := range context.features {
      fmt.Printf("(debug)   feature \"%s\"\n", feature)
    }
    fmt.Printf("(debug)\n")
  }
  fmt.Printf("(debug)\n")
}

// Debug details of a series of scenarios
func debugScenarios(feature *string, scenarios *[]Scenario) {
  fmt.Printf("(debug) *** %s\n", *feature)
  fmt.Printf("(debug)\n")
  for _, scenario := range *scenarios {
    fmt.Printf("(debug) scenario \"%s\":\n", scenario.name)
    for _, step := range scenario.steps {
      fmt.Printf("(debug)  step \"%s\"\n", step)
    }
    fmt.Printf("(debug)\n")
  }
  fmt.Printf("(debug)\n")
}

// Append a context
func appendContext(init *[]Context, parallel *[]Context, end *[]Context, filename string, line string, linenum int) {
  if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
    return
  }

  pair := strings.Split(line, ":")
  if len(pair) != 2 {
    contextsSyntaxError(filename, line, linenum)
  }

  name := strings.TrimSpace(pair[0])
  switch name {
    case "init":
      checkDuplicateContext(init, name, filename, line, linenum)
    case "end":
      checkDuplicateContext(end, name, filename, line, linenum)
    default:
      checkDuplicateContext(parallel, name, filename, line, linenum)
  }

  features := strings.Split(pair[1], ",")
  for i, feature := range features {
    features[i] = strings.TrimSpace(feature)
  }
  switch name {
    case "init":
      *init = append(*init, Context { name: name, features: features })
    case "end":
      *end = append(*end, Context { name: name, features: features })
    default:
      *parallel = append(*parallel, Context { name: name, features: features })
    }
}

// Append a scenario or a step
func appendScenario(feature *string, scenarios *[]Scenario, filename string, line string, linenum int) {
  shortLine := strings.TrimSpace(line)
  if shortLine == "" || strings.HasPrefix(shortLine, "#") {
    return
  }

  if strings.HasPrefix(shortLine, "Feature") {
    *feature = shortLine
    return
  }

  if strings.HasPrefix(shortLine, "Scenario") {
    *scenarios = append(*scenarios, Scenario { name: shortLine, steps: make([]string, 0) })
    return
  }

  if strings.HasPrefix(shortLine, "Given") || strings.HasPrefix(shortLine, "When") || strings.HasPrefix(shortLine, "Then") || strings.HasPrefix(shortLine, "And") {
    if len(*scenarios) > 0 {
      steps := &(*scenarios)[len(*scenarios) - 1].steps
      *steps = append(*steps, shortLine)
      return
    }
  }

  scenariosSyntaxError(filename, line, linenum)
}

//

// Read the scenarios
func readScenarios(feature *string, scenarios *[]Scenario, debug bool, filename string) {
  file, err := os.Open(filename)
  if (err != nil) {
    scenariosReadError(filename, err)
  }
  defer file.Close()

  reader := bufio.NewReader(file)
  buffer := bytes.NewBuffer(make([]byte, 0))

  for linenum := 1; ; linenum++ {
    part, prefix, err := reader.ReadLine()
    if err == io.EOF {
      break
    }
    if err != nil {
      scenariosReadError(filename, err)
    }
    buffer.Write(part)
    if !prefix {
      appendScenario(feature, scenarios, filename, buffer.String(), linenum)
      buffer.Reset()
    }
  }
  if (debug) {
    debugScenarios(feature, scenarios)
  }
}

// Run the scenarios
func runScenarios(feature string, scenarios *[]Scenario, context string) {
  startFeature(context, feature)
  for _, scenario := range *scenarios {
    skip := false
    startScenario(context, scenario.name)
    for _, step := range scenario.steps {
      if skip {
        skipStep(context, step)
      } else {
        if !startStep(context, step) { skip = true }
      }
      // allow a context switch after the step
      time.Sleep(time.Millisecond)
    }
  }
}

// Run all features in a given context
func runFeatures(debug bool, context Context) {
  for _, filename := range context.features {
    feature := ""
    scenarios := []Scenario{}

    readScenarios(&feature, &scenarios, debug, filename)
    runScenarios(feature, &scenarios, context.name)
  }
  wg.Done()
}

// Parse command line parameters
func parseArgs(debug *bool, version *bool, filename *string) {
  for i, arg := range os.Args {
    if i == 0 {
      continue
    }
    if arg == "-d" {
      *debug = true
      continue
    }
    if arg == "-v" {
      *version = true
      continue
    }
    if *filename != "" {
      syntaxError()
    }
    *filename = arg
  }
  if *filename == "" {
    *filename = "features/default.contexts"
  }
}

// Read the contexts
func readContexts(init *[]Context, parallel *[]Context, end *[]Context, debug bool, filename string) {
  file, err := os.Open(filename)
  if (err != nil) {
    contextsReadError(filename, err)
  }
  defer file.Close()

  reader := bufio.NewReader(file)
  buffer := bytes.NewBuffer([]byte{})

  for linenum := 1; ; linenum++ {
    part, prefix, err := reader.ReadLine()
    if err == io.EOF {
      break
    }
    if err != nil {
      contextsReadError(filename, err)
    }
    buffer.Write(part)
    if !prefix {
      appendContext(init, parallel, end, filename, buffer.String(), linenum)
      buffer.Reset()
    }
  }
  if (debug) {
    if len(*init) > 0 {
      debugContexts("Initialization", init)
    }
    debugContexts("Parallel", parallel)
    if len(*end) > 0 {
      debugContexts("End", end)
    }
  }
}

// Run the contexts
func runContexts(init *[]Context, parallel *[]Context, end *[]Context, debug bool) {
  wg.Add(len(*init))
  for _, context := range *init {
    go runFeatures(debug, context)
  }
  wg.Wait()
  wg.Add(len(*parallel))
  for _, context := range *parallel {
    go runFeatures(debug, context)
  }
  wg.Wait()
  wg.Add(len(*end))
  for _, context := range *end {
    go runFeatures(debug, context)
  }
  wg.Wait()
}

// Parallel tests
func ParallelTests() {
  debug := false
  version := false
  filename := ""
  parseArgs(&debug, &version, &filename)
  if version {
    fmt.Printf("paracuke version 0.1\n")
  }

  init := make([]Context, 0)
  parallel := make([]Context, 0)
  end := make([]Context, 0)
  readContexts(&init, &parallel, &end, debug, filename)
  runContexts(&init, &parallel, &end, debug)
}
