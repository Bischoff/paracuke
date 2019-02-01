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

Final results could report file and line number of errors

JUnit reports currently cannot be generated.


## Cucumber facilities

Conditional tags are not supported yet:
```cucumber
@powerpc
  Scenario: Basic arithmetic
```

Similarly, there is no feature nor scenario hooks yet.

There is no function to skip a scenario.

There is no `Background` section.

There is no `Scenario outline` and `Examples`.


## Performance

We could try to read all features before starting to execute them.

Steps regular expressions are currently matched within a linear search.


## Property-based testing

paracuke could be extended to generate tests automatically
and reduce them to a minimal failing sequence:

  https://www.youtube.com/watch?v=hXnS_Xjwk2Y

