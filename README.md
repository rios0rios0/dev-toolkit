# LocalLaunch

LocalLaunch is a CLI tool that reads a YAML slice of code inside a README.md file and executes the commands specified in the up or down section of the YAML file.

## Installation
To install LocalLaunch, please follow the steps below:

1. Clone this repository to your local machine:

```bash
<!-- aaa -->
git clone https://github.com/example/repo.git
```
2. Navigate to the cloned directory:


```bash
cd repo
```
3. Build the executable:


```bash
go build
```
4. Add the executable to your PATH:


```bash
export PATH=$PATH:/path/to/repo
```
## Usage
To use LocalLaunch, you need to specify the path to your README.md file and the command you want to run. LocalLaunch supports two commands: up and down.

Syntax


```bash
lol [README.md] [command]
```
Commands

- `up`: Runs the commands specified in the `up` section of the YAML file.
- `down`: Runs the commands specified in the `down` section of the YAML file.

Example

Let's assume you have a README.md file that contains the following YAML code:

yaml

up:
  - echo "Hello World"
down:
  - echo "Goodbye World"
To run the up command, you would execute the following command:

sh
Copy code
lol README.md up
This would output the following:

sh
Copy code
Hello World
To run the down command, you would execute the following command:

sh
Copy code
lol README.md down
This would output the following:

sh
Copy code
Goodbye World
Note that the commands specified in the YAML file are executed in the order they appear.