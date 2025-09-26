## 🚀 Changes

- Set a default value for the `mechanisms` attribute in the `mongodb_user` resource:
  - **Before:** `Default: setdefault.StaticValue(types.SetNull(types.StringType))`
  - **After:**  
    ```go
    Default: setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{
        types.StringValue("SCRAM-SHA-256"),
    }))
    ```

## 🤖 Behavior

- If `mechanisms` is not explicitly set by the user, it now defaults to `SCRAM-SHA-256`.
- If the user specifies a value, the given mechanism(s) will be used.

## 🛠 Motivation

This change addresses an issue when using the provider with **Amazon DocumentDB**, which does not return the `mechanisms` field in user metadata.  
This caused Terraform to treat the value as "unknown" after apply, resulting in the following error:

```
Error: Provider returned invalid result object after apply

After the apply operation, the provider still indicated an unknown value
for module.<...>.mongodb_user.users["<user>"].mechanisms.
All values must be known after apply, so this is always a bug in the provider...
```

By setting a default (`SCRAM-SHA-256`), the value becomes known, and Terraform no longer considers the resource tainted after apply.

This issue does **not** affect MongoDB, as it correctly returns the `mechanisms` field when querying a user.

## ✅ Compatibility

- ✅ Fully backward-compatible
- ❌ No breaking changes

## 🙏 Thanks

Thanks to the contributor for identifying and fixing this edge case with DocumentDB! 🎉
