import {
  CrudErrorReason,
  isCrudErrorReason,
} from "@servora/proto-utils/error-reasons/servora/crud/v1/errors";

const configuredReason: CrudErrorReason =
  CrudErrorReason.CRUD_ERROR_REASON_INVALID_FILTER;

const reasonMessages: Readonly<Partial<Record<CrudErrorReason, string>>> = {
  [CrudErrorReason.CRUD_ERROR_REASON_INVALID_FILTER]: "筛选条件无效",
};

function narrowReason(value: unknown): CrudErrorReason | undefined {
  if (isCrudErrorReason(value)) {
    const narrowed: CrudErrorReason = value;
    return narrowed;
  }
  return undefined;
}

void configuredReason;
void reasonMessages;
void narrowReason;
