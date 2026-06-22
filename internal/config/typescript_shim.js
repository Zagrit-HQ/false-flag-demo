// Shim for @falseflag/config consumed by the server-side TypeScript
// compiler. The shape mirrors js/packages/config-ts/src/index.ts —
// every builder returns the same plain-object IR node the TypeScript
// authoring package returns on the client. The typescript_conformance
// test exercises both compilers against the same corpus to keep them
// in sync; do not edit one without the other.

"use strict";

function eq(attr, value)        { return { kind: "eq",      attr: attr, value: value }; }
function neq(attr, value)       { return { kind: "neq",     attr: attr, value: value }; }
function isIn(attr, values)     { return { kind: "in",      attr: attr, values: values }; }
function gt(attr, value)        { return { kind: "gt",      attr: attr, value: value }; }
function gte(attr, value)       { return { kind: "gte",     attr: attr, value: value }; }
function lt(attr, value)        { return { kind: "lt",      attr: attr, value: value }; }
function lte(attr, value)       { return { kind: "lte",     attr: attr, value: value }; }
function matches(attr, pattern) { return { kind: "matches", attr: attr, pattern: pattern }; }
function rollout(attr, salt, percent) {
    return { kind: "rollout", attr: attr, salt: salt, percent: percent };
}

function all() {
    var preds = [];
    for (var i = 0; i < arguments.length; i++) { preds.push(arguments[i]); }
    return { kind: "all", of: preds };
}

function any() {
    var preds = [];
    for (var i = 0; i < arguments.length; i++) { preds.push(arguments[i]); }
    return { kind: "any", of: preds };
}

function not(predicate) { return { kind: "not", of_one: predicate }; }
function cel(source)    { return { kind: "cel", source: source }; }
function always()       { return { kind: "always" }; }

function rule(id, when, value) { return { id: id, when: when, value: value }; }

function flag(input) {
    return {
        value_type: input.value_type,
        "default": input["default"],
        rules: input.rules,
    };
}

var FalseFlag = {
    flag: flag, rule: rule,
    eq: eq, neq: neq, in: isIn,
    gt: gt, gte: gte, lt: lt, lte: lte,
    matches: matches, rollout: rollout,
    all: all, any: any, not: not,
    cel: cel, always: always,
};

// Expose the module via a hidden global; the require() stub in
// typescript.go returns this object. __esModule = true tells
// esbuild's interop helper to use the object as-is.
globalThis.__falseflag_dsl = {
    __esModule: true,
    flag: flag, rule: rule,
    eq: eq, neq: neq, in: isIn,
    gt: gt, gte: gte, lt: lt, lte: lte,
    matches: matches, rollout: rollout,
    all: all, any: any, not: not,
    cel: cel, always: always,
    FalseFlag: FalseFlag,
    "default": FalseFlag,
};
