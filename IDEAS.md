# Possible future developments

## Core extensions

Instead of a boolean, the steps could return a value out of three,
meaning: success, failure, or skipped.

There should be a command-line option to ignore one or more contexts:
```
  -i init,end
```
would ignore the special initialization and termination contexts
`init` and `end`.

Instead of 3 phases (init, parallel, and end), we could define as
many as we want.

The syntax of the contexts file could be changed to json, yaml, or XML.
Or mimick the data structures of Go.

There is no final results report printed yet.

JUnit reports currently cannot be generated.

Performance could be increased by reading all features before starting
to execute them.


## Cucumber facilities

Conditional tags are not supported yet:
```cucumber
@powerpc
  Scenario: Basic arithmetic
```

Similarly, there is no feature nor scenario hooks yet.

There is no `Background` section,
nor `Scenario outline` and `Examples`.


## Property-based testing

paracuke could be extended to generate tests automatically
and reduce them to a minimal failing sequence:

  https://www.youtube.com/watch?v=hXnS_Xjwk2Y

