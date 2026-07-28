import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import {
  advancePager,
  applyPager,
  buildFilter,
  buildOrderBy,
  filterContains,
  filterPrefix,
  filterSuffix,
  firstPage,
  makeUpdateMask,
  rawFilterValue,
  ResourceNameError,
} from "../dist/crud/index.js";
import { UserName } from "../dist/gen/servora/example/v1/example.crud.js";

const nameVectors = JSON.parse(
  await readFile(new URL("../../../../conformance/crud/resource_names.json", import.meta.url), "utf8"),
);
const stringMatchVectors = JSON.parse(
  await readFile(new URL("../../../../conformance/crud/string_matches.json", import.meta.url), "utf8"),
);

const fields = {
  DisplayName: "display_name",
  TenantPlan: "tenant_plan",
  CreateTime: "create_time",
};

test("makeUpdateMask preserves generated field order and explicit clears", () => {
  const values = { tenant_plan: undefined, display_name: "Ada" };
  assert.equal(
    makeUpdateMask(["display_name", "tenant_plan"], values),
    "display_name,tenant_plan",
  );
  assert.throws(
    () => makeUpdateMask(["display_name"], { unknown: true }),
    /unknown update field/,
  );
});

test("makeUpdateMask maps generated TypeScript keys to Proto paths", () => {
  const updateFields = {
    displayName: "display_name",
    nickname: "nickname",
  };
  assert.equal(
    makeUpdateMask(updateFields, { nickname: undefined, displayName: "Ada" }),
    "display_name,nickname",
  );
});

test("buildFilter renders typed groups and rejects invalid values", () => {
  assert.equal(
    buildFilter(fields, {
      and: [
        { field: "display_name", operator: "=", value: 'A "quoted" name' },
        {
          field: "tenant_plan",
          operator: "=",
          value: rawFilterValue("ENTERPRISE"),
        },
      ],
    }),
    '(display_name = "A \\"quoted\\" name" AND tenant_plan = ENTERPRISE)',
  );
  assert.throws(
    () =>
      buildFilter(fields, {
        field: "create_time",
        operator: "=",
        value: Number.NaN,
      }),
    /must be finite/,
  );
});

test("buildFilter rejects unknown operators at runtime", () => {
  assert.throws(
    () =>
      buildFilter(fields, {
        field: "display_name",
        operator: "~=",
        value: "foo",
      }),
    RangeError,
  );
});

test("buildFilter matches shared string-match conformance vectors", () => {
  const builders = {
    exact: (value) => value,
    prefix: filterPrefix,
    suffix: filterSuffix,
    contains: filterContains,
    raw: rawFilterValue,
  };
  for (const vector of stringMatchVectors.valid) {
    const value = builders[vector.builder.kind](vector.builder.value);
    assert.equal(
      buildFilter(fields, {
        field: stringMatchVectors.field,
        operator: vector.operator,
        value,
      }),
      vector.wireFilter,
      vector.name,
    );
  }
});

test("typed wildcard helpers are immutable and reject non-equality operators", () => {
  for (const helper of [filterPrefix, filterSuffix, filterContains]) {
    const value = helper("foo");
    assert.ok(Object.isFrozen(value));
    assert.throws(() => {
      value.value = "changed";
    }, TypeError);
    for (const operator of ["<", "<=", ">", ">=", ":"]) {
      assert.throws(
        () =>
          buildFilter(fields, {
            field: "display_name",
            operator,
            value,
          }),
        RangeError,
      );
    }
  }
});

test("buildFilter validates fields while preserving raw filter values", () => {
  assert.throws(
    () =>
      buildFilter(fields, {
        field: "unknown",
        operator: "=",
        value: filterContains("foo"),
      }),
    /unknown filter field/,
  );
  assert.equal(
    buildFilter(fields, {
      field: "display_name",
      operator: "=",
      value: rawFilterValue('"foo*"'),
    }),
    'display_name = "foo*"',
  );
});

test("buildOrderBy emits canonical terms and rejects duplicates", () => {
  assert.equal(
    buildOrderBy(fields, [
      { field: "create_time", direction: "desc" },
      { field: "display_name" },
    ]),
    "create_time desc,display_name",
  );
  assert.throws(
    () =>
      buildOrderBy(fields, [
        { field: "create_time" },
        { field: "create_time" },
      ]),
    /duplicate order field/,
  );
});

test("pager helpers clone requests and terminate on an empty token", () => {
  const request = Object.freeze({ parent: "tenants/acme" });
  assert.deepEqual(applyPager(firstPage, request), request);
  assert.deepEqual(
    applyPager(advancePager("next"), request),
    { parent: "tenants/acme", pageToken: "next" },
  );
  assert.deepEqual(advancePager(""), { exhausted: true });
  assert.deepEqual(request, { parent: "tenants/acme" });
});

test("generated UserName matches the shared Go/TypeScript vectors", () => {
  for (const vector of nameVectors.valid) {
    assert.deepEqual(UserName.parse(vector.name), {
      tenant: vector.tenant,
      user: vector.user,
    });
    assert.equal(
      UserName.format({ tenant: vector.tenant, user: vector.user }),
      vector.name,
    );
  }
  for (const name of nameVectors.invalidNames) {
    assert.throws(() => UserName.parse(name), ResourceNameError);
    assert.equal(UserName.tryParse(name), null);
  }
  for (const parts of nameVectors.invalidParts) {
    assert.throws(() => UserName.format(parts), ResourceNameError);
  }
});
