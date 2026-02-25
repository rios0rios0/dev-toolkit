<h1 align="center">VersaInit</h1>
<p align="center">
    <a href="https://github.com/rios0rios0/versainit/releases/latest">
        <img src="https://img.shields.io/github/release/rios0rios0/versainit.svg?style=for-the-badge&logo=github" alt="Latest Release"/></a>
    <a href="https://github.com/rios0rios0/versainit/blob/main/LICENSE">
        <img src="https://img.shields.io/github/license/rios0rios0/versainit.svg?style=for-the-badge&logo=github" alt="License"/></a>
    <a href="https://github.com/rios0rios0/versainit/actions/workflows/default.yaml">
        <img src="https://img.shields.io/github/actions/workflow/status/rios0rios0/versainit/default.yaml?branch=main&style=for-the-badge&logo=github" alt="Build Status"/></a>
</p>

VersaInit stands for versatile initialization tool. It is a tool written in Go that helps you automatically bootstrap a project by detecting its language and running the appropriate commands.

## Features

- **Automatic Project Initialization**: Quickly set up your project with predefined configurations
- **Multi-language Support**: Supports various programming languages with customizable commands
- **Easy Configuration**: Manage your project setup with a simple YAML configuration file

## Installation

```bash
git clone https://github.com/rios0rios0/versainit.git
cd versainit
go build -o vinit
```

## Configuration

Create a `versainit.yaml` configuration file that defines the languages, their corresponding patterns, and the commands to run for each language:

```yaml
languages:
  python:
    start: 'pdm install && pdm start'
    build: 'pdm build'
    extensions:
      - 'py'
    special_patterns:
      - 'setup.cfg'
      - 'setup.py'
      - 'pyproject.toml'
  java:
    start: 'gradle bootRun'
    build: 'gradle build -x check -x test'
    extensions:
      - 'java'
    special_patterns:
      - 'build.gradle'
      - 'pom.xml'
```

## Usage

Navigate to your project directory and run VersaInit with the following command:

```bash
vinit -c versainit.yaml start
```

For more information, you can run `vinit -h` to see the help message.

## Contributing

Contributions are welcome! Please read the [contributing guidelines](CONTRIBUTING.md) for more details.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
