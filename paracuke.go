// paracuke, a parallel cucumber

package paracuke

import (
  "os"
  "fmt"
  "strings"
  "regexp"
  "sync"
  "time"
  "bufio"
  "bytes"
  "io"
  "io/ioutil"
  "gopkg.in/yaml.v2"
)

// Exported types
type StepFunction func(context *Context, args []string) bool

type Context struct {
  Data map[string]string `yaml:"data"`
  Features []string `yaml:"features"`
}

type ContextWrapper struct {
  ContextWrapper Context `yaml:"context"`
}

type Batch struct {
  BatchWrapper []ContextWrapper `yaml:"batch"`
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
func contextsSyntaxError(filename string, err error) {
  fmt.Fprintf(os.Stderr, "\x1b[31mSyntax error in contexts file \"%s\":\x1b[30m\n", filename)
  fmt.Fprintf(os.Stderr, "\x1b[31m  %s\x1b[30m\n", err.Error())
  os.Exit(2)
}

// Missing context name error
func missingContextNameError(b1 int, c1 int, filename string) {
  fmt.Fprintf(os.Stderr, "\x1b[31mMissing context name in contexts file \"%s\":\x1b[30m\n", filename)
  fmt.Fprintf(os.Stderr, "\x1b[31m  batch #%d, context #%d\x1b[30m\n", b1 + 1, c1 + 1)
  os.Exit(3)
}

// Duplicate context name error
func duplicateContextNameError(filename string, name string) {
  fmt.Fprintf(os.Stderr, "\x1b[31mDuplicate context name in contexts file \"%s\":\x1b[30m\n", filename)
  fmt.Fprintf(os.Stderr, "\x1b[31m  \"%s\"\x1b[30m\n", name)
  os.Exit(4)
}

// Error reading the feature
func featureReadError(filename string, err error) {
  fmt.Fprintf(os.Stderr, "\x1b[31mUnable to read feature file \"%s\":\x1b[30m\n", filename)
  fmt.Fprintf(os.Stderr, "\x1b[31m  %s\x1b[30m\n", err.Error())
  os.Exit(5)
}

// Syntax error in the feature's lines
func lineSyntaxError(filename string, line string, linenum int) {
  fmt.Fprintf(os.Stderr, "\x1b[31mSyntax error on line %d of feature file \"%s\":\x1b[30m\n", linenum, filename)
  fmt.Fprintf(os.Stderr, "\x1b[31m  \"%s\"\x1b[30m\n", line)
  os.Exit(6)
}

// Check name of context
func checkContextName(batches *[]Batch, context *Context, b1 int, c1 int, filename string) {
  name := context.Data["name"]
  // existence
  if name == "" {
    missingContextNameError(b1, c1, filename)
  }
  // unicity
  for b2, batch := range *batches {
    for c2, other := range batch.BatchWrapper {
      if b2 != b1 || c2 != c1 {
        otherContext := &other.ContextWrapper
        otherName := otherContext.Data["name"]
        if otherName == name {
          duplicateContextNameError(filename, name)
        }
      }
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

// Check batches after reading them
func checkBatches(batches *[]Batch, filename string) {
  for b1, batch := range *batches {
    for c1, context := range batch.BatchWrapper {
      checkContextName(batches, &context.ContextWrapper, b1, c1, filename)
    }
  }
}

// Debug details of a series of batches
func debugBatches(batches *[]Batch) {
  for _, batch := range *batches {
    fmt.Printf("(debug) batch\n")
    fmt.Printf("(debug)\n")
    for _, context := range batch.BatchWrapper {
      fmt.Printf("(debug)   context \"%s\":\n", context.ContextWrapper.Data["name"])
      for _, feature := range context.ContextWrapper.Features {
        fmt.Printf("(debug)     feature \"%s\"\n", feature)
      }
      fmt.Printf("(debug)\n")
    }
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

  fmt.Printf("(results)\n")
  if len(run.failedScenarios) > 0 {
    fmt.Printf("(results)  \x1b[31mFailed scenarios:\x1b[30m\n")
    for _, fail := range run.failedScenarios {
      fmt.Printf("(results)  \x1b[31m(%s)  %s\x1b[30m\n", fail.ctxt, fail.scen)
    }
    fmt.Printf("(results)  \n")
  }

  fmt.Printf("(results)  %d scenarios (\x1b[32m%d successful\x1b[30m, \x1b[31m%d failed\x1b[30m, \x1b[36m%d skipped\x1b[30m)\n",
             totalScenarios, run.successfulScenarios, len(run.failedScenarios), run.skippedScenarios)
  fmt.Printf("(results)  %d steps (\x1b[32m%d successful\x1b[30m, \x1b[31m%d failed\x1b[30m, \x1b[36m%d skipped\x1b[30m)\n",
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

// Read the batches
func readBatches(batches *[]Batch, debug bool, filename string) {
  yamlFile, err := ioutil.ReadFile(filename)
  if err != nil {
    contextsReadError(filename, err)
  }
  err = yaml.UnmarshalStrict(yamlFile, batches)
  if err != nil {
    contextsSyntaxError(filename, err)
  }
  checkBatches(batches, filename)
  if (debug) {
    debugBatches(batches)
  }
}

// Run the batches
func runBatches(batches *[]Batch, debug bool) {
  run := cucumberRun {
    debug,
    0, []failure { }, 0,  // scenario statistics
    0, 0, 0,              // step statistics
  }

  for _, batch := range *batches {
    wg.Add(len(batch.BatchWrapper))
    for _, context := range batch.BatchWrapper {
      go runFeatures(&run, context.ContextWrapper)
    }
    wg.Wait()
  }
  reportResults(&run)
}

// Run tests
func RunTests() {
  debug := false
  version := false
  filename := ""
  parseArgs(&debug, &version, &filename)
  if version {
    fmt.Printf("(version)  paracuke version 0.2\n")
  }

  conf := []Batch { }
  readBatches(&conf, debug, filename)
  runBatches(&conf, debug)
}
