local images = import 'images.libsonnet';
local schemas = import 'schemas.libsonnet';

local cache = std.native('cache');
local copy = std.native('copy');
local copyFrom = std.native('copyFrom');

// External cache for go compiler, go mod, golangci-lint
local gocache = [
    cache("go-build", "/app/cache"),
    cache("go-mod", "/go/pkg/mod"),
];

// Sources which will be tracked for changes
local gosources = [
    "go.mod",
    "go.sum",
    "cmd",
    "api",
    "pkg",
];

{
    // Function that generate project build definitions, including code generating, app compilation, etc.
    project(appIDs, openAPIServers):: {
        apiVersion: "brewkit/v1",

        targets: {
            all: ['build', 'check'],

            // build target to chain all build of apps
            build: [appID for appID in appIDs],

            gobase: {
                from: images.gocompiler,
                workdir: "/app",
                env: {
                    GOCACHE: "/app/cache/go-build",
                },
                copy: copyFrom(
                    'gosources',
                    '/app',
                    '/app'
                ),
            },
        } + {
            [appID]: {
                from: "gobase",
                workdir: "/app",
                cache: gocache,
                dependsOn: ['generate', 'modules'],
                command: 'go build \\
                        -trimpath -v \\
                        -o ./bin/' + appID + ' ./cmd/' + appID,
                output: {
                    artifact: "/app/bin/" + appID,
                    "local": "./bin",
                },
            }
            for appID in appIDs // expand build target for each appID
        } + {
            [appID + "debug"]: {
                from: "gobase",
                workdir: "/app",
                cache: gocache,
                dependsOn: ['generate', 'modules'],
                command: 'go build \\
                        -trimpath -v \\
                        --gcflags="all=-N -l" \\
                        -o ./bin/' + appID + ' ./cmd/' + appID,
                output: {
                    artifact: "/app/bin/" + appID,
                    "local": "./bin",
                },
            }
            for appID in appIDs // expand debug build target for each appID
        } + {
            gosources: { // layer for go source code
                from: "scratch",
                workdir: "/app",
                copy: [copy(source, source) for source in gosources]
            },

            generate: [
                'generateopenapiservers',
            ],

            generateopenapiservers: schemas.generateOpenApiServers(openAPIServers),

            modules: ["gotidy"],

            gotidy: {
                from: "gobase",
                workdir: "/app",
                cache: gocache,
                ssh: {},
                command: "go mod tidy",
                output: {
                    artifact: "/app/go.*",
                    "local": ".",
                },
            },

            // export local copy of dependencies for ide index
            modulesvendor: {
                from: "gotidy",
                workdir: "/app",
                cache: gocache,
                command: "go mod vendor",
                output: {
                    artifact: "/app/vendor",
                    "local": "vendor",
                },
            },

            check: {
                from: images.golangcilint,
                workdir: "/app",
                env: {
                    GOCACHE: "/app/cache/go-build",
                    GOLANGCI_LINT_CACHE: "/app/cache/go-build",
                },
                cache: gocache,
                copy: [
                    copy('.golangci.yml', '.golangci.yml'),
                    copyFrom(
                        'gosources',
                        '/app',
                        '/app'
                    ),
                ],
                command: "golangci-lint run",
            },
        },
    },
}
