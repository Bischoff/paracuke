// paracuke, a parallel cucumber

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

// Exported types
type StepFunction func(context *Context, args []string) bool

type Context struct {
  Data map[string]string
  Features []string
}

// Internal types
type registeredStep struct {
  re *regexp.Regexp
  step StepFunction
}

type scenario struct {
  title string
  steps []string
}

type feature struct {
  title string
  description []string
  scenarios []scenario
}

type failure struct {
  ctxt string
  scen string
}

type cucumberRun struct {
  debug bool
  successfulScenarios int
  failedScenarios []failure
  skippedScenarios int
  successfulSteps int
  failedSteps int
  skippedSteps int
}

// Global variables
var registeredSteps []registeredStep = []registeredStep{}

var wg sync.WaitGroup

// Cucumber steps definition
func registerStep(reStr string, stepFunc StepFunction) {
  registeredSteps = append(registeredSteps, registeredStep { re: regexp.MustCompile(reStr), step: stepFunc })
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

// Syntax error
func syntaxError() {
  fmt.Fprintf(os.Stderr, "Syntax: %s [-v|-d] [<contexts>]\n", os.Args[0])
  fmt.Fprintf(os.Stderr, "  -v: show version\n")
  fmt.Fprintf(os.Stderr, "  -d: debug mode\n")
  fmt.Fprintf(os.Stderr, "  contexts: test contexts file\n")
  os.Exit(1)
}

// Error reading the contexts
func contextsReadError(filename string, err error) {
  fmt.Fprintf(os.Stderr, "\x1b[31mUnable to read contexts file \"%s\":\n", filename)
  fmt.Fprintf(os.Stderr, "\x1b[31m  %s\x1b[30m\n", err.Error())
  os.Exit(2)
}

// Syntax error in the contexts
func contextsSyntaxError(filename string, line string, linenum int) {
  fmt.Fprintf(os.Stderr, "\x1b[31mSyntax error on line %d of contexts file \"%s\":\x1b[30m\n", linenum, filename)
  fmt.Fprintf(os.Stderr, "\x1b[31m  \"%s\"\x1b[30m\n", line)
  os.Exit(2)
}

// Duplicate context name error
func duplicateContextError(filename string, line string, linenum int) {
  fmt.Fprintf(os.Stderr, "\x1b[31mDuplicate context name on line %d of contexts file \"%s\":\x1b[30m\n", linenum, filename)
  fmt.Fprintf(os.Stderr, "\x1b[31m  \"%s\"\x1b[30m\n", line)
  os.Exit(3)
}

// Error reading the feature
func featureReadError(filename string, err error) {
  fmt.Fprintf(os.Stderr, "\x1b[31mUnable to read feature file \"%s\":\x1b[30m\n", filename)
  fmt.Fprintf(os.Stderr, "\x1b[31m  %s\x1b[30m\n", err.Error())
  os.Exit(4)
}

// Syntax error in the feature's lines
func lineSyntaxError(filename string, line string, linenum int) {
  fmt.Fprintf(os.Stderr, "\x1b[31mSyntax error on line %d of feature file \"%s\":\x1b[30m\n", linenum, filename)
  fmt.Fprintf(os.Stderr, "\x1b[31m  \"%s\"\x1b[30m\n", line)
  os.Exit(5)
}

// Check duplicate context name
func checkDuplicateContext(contexts *[]Context, name string, filename string, line string, linenum int) {
  for _, context := range *contexts {
    if (context.Data["name"] == name) {
      duplicateContextError(filename, line, linenum)
    }
  }
}

// Execute a step
func executeStep(run *cucumberRun, context *Context, stepTitle string, step StepFunction, args []string) bool {
  name := context.Data["name"]

  if step(context, args) {
    run.successfulSteps++
    fmt.Printf("\x1b[32m(%s)    %s\x1b[30m\n", name, stepTitle)
    return true
  }
  run.failedSteps++
  fmt.Printf("\x1b[31m(%s)    %s\x1b[30m\n", name, stepTitle)
  fmt.Printf("\x1b[31m(%s)    Step failed!\x1b[30m\n", name)
  return false
}

// Start a feature
func startFeature(context *Context, feat *feature) {
  name := context.Data["name"]

  fmt.Printf("(%s)  %s\n", name, feat.title)
  fmt.Printf("(%s)\n", name)
  for _, desc := range feat.description {
    fmt.Printf("(%s)    %s\n", name, desc)
  }
  fmt.Printf("(%s)\n", name)
}

// Start a scenario
func startScenario(context *Context, scen *scenario) {
  name := context.Data["name"]

  fmt.Printf("(%s)  %s\n", name, scen.title)
}

// Skip a step
func skipStep(run *cucumberRun, context *Context, stepTitle string) {
  run.skippedSteps++
  name := context.Data["name"]
  fmt.Printf("\x1b[36m(%s)    %s\x1b[30m\n", name, stepTitle)
  fmt.Printf("\x1b[36m(%s)      (skipped...)\x1b[30m\n", name)
}

// Start a step
func startStep(run *cucumberRun, context *Context, stepTitle string) bool {
  stepPrefix := ""
  for _, stepPrefix = range []string { "Given", "When", "Then", "And" } {
    if strings.HasPrefix(stepTitle, stepPrefix) { break }
  }
  toMatch := strings.TrimSpace(strings.TrimPrefix(stepTitle, stepPrefix))
  for _, reg := range registeredSteps {
    if reg.re.MatchString(toMatch) {
      args := reg.re.FindStringSubmatch(toMatch)
      return executeStep(run, context, stepTitle, reg.step, args)
    }
  }
  fmt.Printf("\n\x1b[31m(%s) *** Please implement step:\n   \"%s\"\x1b[30m\n\n", context, stepTitle)
  return false
}

// Debug details of a series of contexts
func debugContexts(title string, contexts *[]Context) {
  fmt.Printf("(debug) section \"%s\"\n", title)
  fmt.Printf("(debug)\n")
  for _, context := range *contexts {
    fmt.Printf("(debug)   context \"%s\":\n", context.Data["name"])
    for _, feature := range context.Features {
      fmt.Printf("(debug)     feature \"%s\"\n", feature)
    }
    fmt.Printf("(debug)\n")
  }
  fmt.Printf("(debug)\n")
}

// Debug details of a feature
func debugFeature(feat *feature) {
  fmt.Printf("(debug) feature \"%s\"\n", feat.title)
  for _, desc := range feat.description {
    fmt.Printf("(debug)   description \"%s\"\n", desc)
  }
  fmt.Printf("(debug)\n")
  for _, scen := range feat.scenarios {
    fmt.Printf("(debug)   scenario \"%s\":\n", scen.title)
    for _, step := range scen.steps {
      fmt.Printf("(debug)     step \"%s\"\n", step)
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
  context := Context { Data: make(map[string]string), Features: features }
  context.Data["name"] = name
  switch name {
    case "init":
      *init = append(*init, context)
    case "end":
      *end = append(*end, context)
    default:
      *parallel = append(*parallel, context)
    }
}

// Append a line to a feature
func appendLine(line string, feat *feature) bool {
  shortLine := strings.TrimSpace(line)
  if shortLine == "" || strings.HasPrefix(shortLine, "#") {
    return true
  }

  // Feature title
  if strings.HasPrefix(shortLine, "Feature:") {
    if len(feat.scenarios) != 0 { return false }
    feat.title = shortLine
    return true
  }

  // Scenario title
  if strings.HasPrefix(shortLine, "Scenario:") {
    feat.scenarios = append(feat.scenarios, scenario { title: shortLine, steps: make([]string, 0) } )
    return true
  }

  // Step
  for _, stepPrefix := range []string { "Given", "When", "Then", "And" } {
    if strings.HasPrefix(shortLine, stepPrefix) {
      if len(feat.scenarios) == 0 { return false }
      lastScenario := &feat.scenarios[len(feat.scenarios) - 1]
      lastScenario.steps = append(lastScenario.steps, shortLine)
      return true
    }
  }

  // Feature description
  if len(feat.scenarios) != 0 { return false }
  feat.description = append(feat.description, shortLine)
  return true
}

// Read the feature
func readFeature(filename string, run *cucumberRun, feat *feature) {
  file, err := os.Open(filename)
  if (err != nil) {
    featureReadError(filename, err)
    return
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
      featureReadError(filename, err)
      return
    }
    buffer.Write(part)
    if !prefix {
      if !appendLine(buffer.String(), feat) {
        lineSyntaxError(filename, buffer.String(), linenum)
      }
      buffer.Reset()
    }
  }
  if (run.debug) {
    debugFeature(feat)
  }
}

// Run a feature
func runFeature(run *cucumberRun, context *Context, feat *feature) {
  startFeature(context, feat)
  for _, scen := range feat.scenarios {
    failed := false
    startScenario(context, &scen)
    for _, step := range scen.steps {
      if failed {
        skipStep(run, context, step)
      } else {
        if !startStep(run, context, step) { failed = true }
      }
      // allow a context switch after the step
      time.Sleep(time.Millisecond)
    }
    if failed {
      run.failedScenarios = append(run.failedScenarios, failure { context.Data["name"], scen.title } )
    } else {
      run.successfulScenarios++
    }
    fmt.Printf("(%s)\n", context.Data["name"])
  }
}

// Run all features in a given context
// - "run" is passed by pointer, because it is global to all goroutines
// - "context" is passed by value, because it is local to a goroutine
func runFeatures(run *cucumberRun, context Context) {
  for _, filename := range context.Features {
    feat := feature { "", []string{ }, []scenario{ } }

    readFeature(filename, run, &feat)
    runFeature(run, &context, &feat)
  }
  wg.Done()
}

// Report the global results
func reportResults(run *cucumberRun) {
  totalScenarios := run.successfulScenarios + len(run.failedScenarios) + run.skippedScenarios
  totalSteps := run.successfulSteps + run.failedSteps + run.skippedSteps

  fmt.Printf("\n")
  if len(run.failedScenarios) > 0 {
    fmt.Printf("\x1b[31mFailed scenarios:\x1b[30m\n")
    for _, fail := range run.failedScenarios {
      fmt.Printf("\x1b[31m(%s)  %s\x1b[30m\n", fail.ctxt, fail.scen)
    }
    fmt.Printf("\n")
  }

  fmt.Printf("%d scenarios (\x1b[32m%d successful\x1b[30m, \x1b[31m%d failed\x1b[30m, \x1b[36m%d skipped\x1b[30m)\n",
             totalScenarios, run.successfulScenarios, len(run.failedScenarios), run.skippedScenarios)
  fmt.Printf("%d steps (\x1b[32m%d successful\x1b[30m, \x1b[31m%d failed\x1b[30m, \x1b[36m%d skipped\x1b[30m)\n",
             totalSteps, run.successfulSteps, run.failedSteps, run.skippedSteps)
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
  run := cucumberRun {
    debug,
    0, []failure { }, 0,  // scenario statistics
    0, 0, 0,               // step statistics
  }

  wg.Add(len(*init))
  for _, context := range *init {
    go runFeatures(&run, context)
  }
  wg.Wait()
  wg.Add(len(*parallel))
  for _, context := range *parallel {
    go runFeatures(&run, context)
  }
  wg.Wait()
  wg.Add(len(*end))
  for _, context := range *end {
    go runFeatures(&run, context)
  }
  wg.Wait()
  reportResults(&run)
}

// Run tests
func RunTests() {
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
