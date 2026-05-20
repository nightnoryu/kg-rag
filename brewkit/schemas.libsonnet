local images = import 'images.libsonnet';

local copy = std.native('copy');
local copyFrom = std.native('copyFrom');

local serversPrefix = 'api/server/';

{
    generateOpenApiServers(openAPIs): {
        local openAPIDirectoryPath(name) = serversPrefix + name,
        local openAPIFilePath(name) = serversPrefix + name + '.yaml',
        local openAPIConfigFilePath(name) = serversPrefix + name + '/.ogen.yml',

        local oapicodegenCommand(name) =
            'ogen --target ' + openAPIDirectoryPath(name) +
            ' --package ' + name +
            ' --config ' + openAPIConfigFilePath(name) +
            ' --clean ' + openAPIFilePath(name),

        local mappedFiles = [
            copy(openAPIFilePath(name), openAPIFilePath(name)) for name in openAPIs
        ] + [
            copy(openAPIConfigFilePath(name), openAPIConfigFilePath(name)) for name in openAPIs
        ],
        local generateCommands = [oapicodegenCommand(openAPI) for openAPI in openAPIs],

        from: images.gocompiler,
        workdir: "/app",
        copy: mappedFiles + [
            // copy ogen into builder image
            copyFrom(
                images.oapicodegen,
                "/ogen",
                "/usr/local/bin/ogen"
            ),
        ],
        command: std.join(' && ', generateCommands),
        output: {
            artifact: "/app/api",
            "local": "./api"
        },
    },
}
