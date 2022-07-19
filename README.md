# Yacht

[![GoPkg Widget](https://pkg.go.dev/badge/github.com/dixudx/yacht.svg)](https://pkg.go.dev/github.com/dixudx/yacht)
[![License](https://img.shields.io/github/license/dixudx/yacht)](https://www.apache.org/licenses/LICENSE-2.0.html)
![GoVersion](https://img.shields.io/github/go-mod/go-version/dixudx/yacht)
[![Go Report Card](https://goreportcard.com/badge/github.com/dixudx/yacht)](https://goreportcard.com/report/github.com/dixudx/yacht)
![build](https://github.com/dixudx/yacht/actions/workflows/ci.yml/badge.svg)
[![Version](https://img.shields.io/github/v/release/dixudx/yacht)](https://github.com/dixudx/yacht/releases)
[![codecov](https://codecov.io/gh/dixudx/yacht/branch/main/graph/badge.svg)](https://codecov.io/gh/dixudx/yacht)

---

Light-weighted Kubernetes controller-runtime Framework with Minimal Dependencies

---

## Why Building Yacht

Well, there are quite a few controller/operator frameworks out there, such
as [kubebuilder](https://github.com/kubernetes-sigs/kubebuilder),
[operator-sdk](https://github.com/operator-framework/operator-sdk),
[controller-runtime](https://github.com/kubernetes-sigs/controller-runtime). But they are not quite handy.

First of all, building a scaffold project may not be what we really need. Most of the time, we are not trying to build a
project from scratch, but rather adding new controllers on an existing project. Moreover, the structure of the scaffold
project does not suit all. It is not easy to customize the scaffolds according to our own preferences.

Secondly, **backwards compatibility**. Most frameworks **DO NOT** guarantee any particular compatibility matrix between
kubernetes library dependencies ([client-go](https://github.com/kubernetes/client-go),
[apimachinery](https://github.com/kubernetes/apimachinery), etc). This is painful when we want to pin Kubernetes library
dependencies to any specific lower versions. It has always been a headache to upgrade/downgrade the versions of
frameworks and Kubernetes library dependencies.

Last but not least, we **DO** want a light-weighted framework with minimal dependencies.

That's why I build [yacht](https://github.com/dixudx/yacht), which brings pleasure user experiences on writing
Kubernetes controllers. It is built with minimal dependencies, which makes it easier to downgrade/upgrade module
versions.

## To Start Using Yacht

1. Add `github.com/dixudx/yacht` to your `go.mod` file.
2. Follow [yacht examples](./examples) and learn how to use.
