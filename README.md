# paracuke 0.2
<p align="center"><img src=assets/images/Paracuke.png></p>
paracuke is a parallel cucumber written in go

  https://docs.cucumber.io

so that high testing performance is guaranteed even
with slow tests that imply waiting a lot.

It might also be used to stress a tested program
with many parallel requests.

The parallelism resides at the level of the cucumber
engine, freeing the tester from the burden of
implementing it inside the step definitions.

 * [license](LICENSE)
 * [reference documentation](documentation/REFERENCE.md)
 * [possible evolutions](documentation/IDEAS.md)
 * [example contexts file](documentation/examples/example.yml)
