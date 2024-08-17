# VersaInit

VersaInit stands for versatile initialization tool. It is a tool written in Go that helps you automatically bootstrap a project.

## Features

- **Automatic Project Initialization**: Quickly set up your project with predefined configurations.
- **Multi-language Support**: Supports various programming languages with customizable commands.
- **Easy Configuration**: Manage your project setup with a simple YAML configuration file.

## Prerequisites

- **Go**: Ensure you have Go installed on your system. You can download it from [here](https://golang.org/dl/).

## Installation

Clone the repository and build the project using the following commands:

```bash
git clone https://github.com/yourusername/versainit.git
cd versainit
go build
```

## Usage

To use VersaInit, you need a configuration file. The configuration file is a YAML file that contains the languages, their corresponding patterns, and the commands to run for each language. Here is an example of a configuration file:

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
    start: "gradle bootRun"
    build: "gradle build -x check -x test"
    extensions:
      - "java"
    special_patterns:
      - "build.gradle"
      - "pom.xml"
```

Navigate to your project directory and run VersaInit with the following command:

```bash
vinit -c versainit.yaml start
```

For more information, you can run `vinit -h` to see the help message.

## Contributing

We welcome contributions! Please read our [contributing guidelines](CONTRIBUTING.md) for more details.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
