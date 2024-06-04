# ZTMF

## Overview

This project contains the application's UI built with React, Vite, TypeScript, and SWC. It requires Node.js v20, [nvm](https://github.com/nvm-sh/nvm) (or [n](https://github.com/tj/n)), and Yarn v4.

## System Requirements

- [Node 20](https://nodejs.org/en/download)
- [Yarn](https://yarnpkg.com/getting-started/install)
- [GitLeaks](https://github.com/gitleaks/gitleaks/tree/master#installing)

## Getting Started

1. Clone the repository and `cd` into the root directory:

```shell
git clone git@github.com:aquia/template-vite-react`

cd template-vite-react
```

2. Setup Node.js and Yarn

```shell
# using nvm
nvm install --latest-npm
nvm use

# using n
n install auto
n use auto

# enable corepack
corepack enable yarn
```

3. Install dependencies:

```shell
yarn
```

3. Install pre-commit hooks:

```shell
yarn prepare
```

## Building

To build the application, run the following from the root directory:

```shell
yarn build
```

## Testing

To run all tests, run the following from the root directory:

```shell
yarn test
```

To lint all files, run the following from the root directory:

```shell
yarn lint
```

## Developing

First, run the post-install script to create the local development environment file from the example environment file.

```shell
sh ./scripts/post-install.sh
```

To start the local development server, run the following from the root directory:

```shell
yarn dev
```
