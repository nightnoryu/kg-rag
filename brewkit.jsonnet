local project = import 'brewkit/project.libsonnet';

local appIDs = [
    'rag-server'
];

local openAPIServers = [
    'ragapi',
];

project.project(appIDs, openAPIServers)
