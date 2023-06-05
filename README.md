# VersaInit

VersaInit stands for versatile initialization tool. It is a tool written in Go that can help you to automatically bootstrap a project.

## Building

To build the project, you need to have Go installed. Then, you can run the following command:

```bash
go build
```

## Usage

To use the tool, you need to have a configuration file. The configuration file is a YAML file that contains the languages, their corresnponding patterns, and the commands to run for each language. Here is an example of a configuration file:

```yaml
languages:
  python:
    start: "pdm install && pdm start"
    build: "pdm build"
    extensions:
      - "py"
    special_patterns:
      - "setup.cfg"
      - "setup.py"
      - "pyproject.toml"
  java:
    start: gradle bootRun
    build: gradle build -x check -x test
    extensions:
      - "java"
    special_patterns:
      - "build.gradle"
      - "pom.xml"
```

Then, use `cd` to navigate to the directory of your project and run VersaInit. Below is an example of starting the program with VersaInit:

```bash
vinit -c versainit.yaml start
```

For more information, you can run `vinit -h` to see the help message.
