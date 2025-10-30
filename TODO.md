Things To DO:

- Add bash auto completion
  - Basic implementation done, need to add to install script
- Add a way to exclude or blacklist certain directories/paths.
  - Example, excluding the `vendor` directory in a golang project:  findref -e|--exclude  vendor 'some.golang.variable'
  - Example, excluding the `build` directory in a typescript project:  findref -e|--exclude  build 'some.typescript.variable'
  - Example, excluding the `node_modules` directory in a javascript project:  findref -e|--exclude  node_modules 'some.javascript.variable'
- Security and code scanning tools
