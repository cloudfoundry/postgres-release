#!/bin/bash -exu

ROOT="${PWD}"

getent hosts api.apps.pgci-cf.microbosh
echo "getent finished with status : $?"
