import assert from "node:assert/strict";
import test from "node:test";

import {
  CrudErrorReason,
  isCrudErrorReason,
} from "@servora/proto-utils/error-reasons/servora/crud/v1/errors";

const expectedReasons = {
  CRUD_ERROR_REASON_UNSPECIFIED: "CRUD_ERROR_REASON_UNSPECIFIED",
  CRUD_ERROR_REASON_INVALID_RESOURCE_NAME:
    "CRUD_ERROR_REASON_INVALID_RESOURCE_NAME",
  CRUD_ERROR_REASON_INVALID_PAGE_TOKEN: "CRUD_ERROR_REASON_INVALID_PAGE_TOKEN",
  CRUD_ERROR_REASON_INVALID_FILTER: "CRUD_ERROR_REASON_INVALID_FILTER",
  CRUD_ERROR_REASON_INVALID_ORDER_BY: "CRUD_ERROR_REASON_INVALID_ORDER_BY",
  CRUD_ERROR_REASON_INVALID_FIELD_MASK: "CRUD_ERROR_REASON_INVALID_FIELD_MASK",
  CRUD_ERROR_REASON_INVALID_FIELD_VALUE: "CRUD_ERROR_REASON_INVALID_FIELD_VALUE",
  CRUD_ERROR_REASON_INTERNAL: "CRUD_ERROR_REASON_INTERNAL",
};

test("error reason subpath exposes the complete runtime contract", () => {
  assert.deepEqual(CrudErrorReason, expectedReasons);

  for (const reason of Object.values(expectedReasons)) {
    assert.equal(isCrudErrorReason(reason), true);
  }
});

test("error reason guard rejects non-members and prototype properties", () => {
  const invalidReasons = [
    "CRUD_ERROR_REASON_UNKNOWN",
    "crud_error_reason_internal",
    "toString",
    null,
    500,
    {},
  ];

  for (const reason of invalidReasons) {
    assert.equal(isCrudErrorReason(reason), false);
  }
});

test("generated internals remain hidden behind the semantic export", async () => {
  const hiddenPaths = [
    "@servora/proto-utils/dist/gen/servora/crud/v1/errors.errors.js",
    "@servora/proto-utils/src/gen/servora/crud/v1/errors.errors.ts",
  ];

  for (const specifier of hiddenPaths) {
    await assert.rejects(import(specifier), (error) => {
      assert.equal(error.code, "ERR_PACKAGE_PATH_NOT_EXPORTED");
      return true;
    });
  }
});

test("existing public subpaths remain resolvable", async () => {
  const [root, crud, errors, proto] = await Promise.all([
    import("@servora/proto-utils"),
    import("@servora/proto-utils/crud"),
    import("@servora/proto-utils/errors"),
    import("@servora/proto-utils/proto/servora/crud/v1"),
  ]);

  assert.equal(typeof root, "object");
  assert.equal(typeof crud.makeUpdateMask, "function");
  assert.equal(typeof errors.ApiError, "function");
  assert.equal(typeof proto, "object");
});
