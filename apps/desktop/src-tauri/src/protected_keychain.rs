use serde::Serialize;

const PROTECTED_SERVICE: &str = "local.chronosync.signing.user-presence.v1";

#[derive(Clone, Copy, Debug, PartialEq, Eq, Serialize)]
#[serde(rename_all = "snake_case")]
pub enum AuthorizationOutcome {
    Signed,
    UserPresenceRequired,
    AuthorizationCancelled,
    AuthorizationDenied,
    KeyMissing,
    IdentityMismatch,
    UnsupportedAccessControl,
    UnsupportedPlatform,
    MalformedRequest,
    InternalError,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum ProbeOutcome {
    ProtectedItemCreated,
    UnsupportedAccessControl(i32),
    #[cfg_attr(target_os = "macos", allow(dead_code))]
    UnsupportedPlatform,
    MalformedRequest,
    InternalError,
}

pub fn map_security_status(code: i32) -> AuthorizationOutcome {
    match code {
        -128 => AuthorizationOutcome::AuthorizationCancelled,
        -25293 => AuthorizationOutcome::AuthorizationDenied,
        -25308 => AuthorizationOutcome::UserPresenceRequired,
        -25300 => AuthorizationOutcome::KeyMissing,
        -34018 | -50 => AuthorizationOutcome::UnsupportedAccessControl,
        _ => AuthorizationOutcome::InternalError,
    }
}

#[cfg(target_os = "macos")]
fn protected_options(
    account: &str,
    include_access_control: bool,
) -> Result<security_framework::passwords::PasswordOptions, AuthorizationOutcome> {
    use security_framework::access_control::{ProtectionMode, SecAccessControl};
    use security_framework::passwords::{AccessControlOptions, PasswordOptions};

    if account.len() != 64 || !account.bytes().all(|value| value.is_ascii_hexdigit()) {
        return Err(AuthorizationOutcome::MalformedRequest);
    }
    let mut options = PasswordOptions::new_generic_password(PROTECTED_SERVICE, account);
    options.use_protected_keychain();
    options.set_access_synchronized(Some(false));
    if include_access_control {
        let access = SecAccessControl::create_with_protection(
            Some(ProtectionMode::AccessibleWhenUnlockedThisDeviceOnly),
            AccessControlOptions::USER_PRESENCE.bits(),
        )
        .map_err(|error| map_security_status(error.code()))?;
        options.set_access_control(access);
    }
    Ok(options)
}

#[cfg(target_os = "macos")]
pub fn probe_protected_creation(account: &str, seed: &[u8]) -> ProbeOutcome {
    if seed.len() != 32 {
        return ProbeOutcome::MalformedRequest;
    }
    let options = match protected_options(account, true) {
        Ok(options) => options,
        Err(AuthorizationOutcome::MalformedRequest) => return ProbeOutcome::MalformedRequest,
        Err(AuthorizationOutcome::UnsupportedAccessControl) => {
            return ProbeOutcome::UnsupportedAccessControl(-50)
        }
        Err(_) => return ProbeOutcome::InternalError,
    };
    match security_framework::passwords::set_generic_password_options(seed, options) {
        Ok(()) => {
            let delete = protected_options(account, false).and_then(|query| {
                security_framework::passwords::delete_generic_password_options(query)
                    .map_err(|error| map_security_status(error.code()))
            });
            if delete.is_ok() {
                ProbeOutcome::ProtectedItemCreated
            } else {
                ProbeOutcome::InternalError
            }
        }
        Err(error) => match error.code() {
            code @ (-34018 | -50) => ProbeOutcome::UnsupportedAccessControl(code),
            _ => ProbeOutcome::InternalError,
        },
    }
}

#[cfg(not(target_os = "macos"))]
pub fn probe_protected_creation(_: &str, _: &[u8]) -> ProbeOutcome {
    ProbeOutcome::UnsupportedPlatform
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn security_statuses_are_typed_and_closed() {
        let contract = [
            AuthorizationOutcome::Signed,
            AuthorizationOutcome::UserPresenceRequired,
            AuthorizationOutcome::AuthorizationCancelled,
            AuthorizationOutcome::AuthorizationDenied,
            AuthorizationOutcome::KeyMissing,
            AuthorizationOutcome::IdentityMismatch,
            AuthorizationOutcome::UnsupportedAccessControl,
            AuthorizationOutcome::UnsupportedPlatform,
            AuthorizationOutcome::MalformedRequest,
            AuthorizationOutcome::InternalError,
        ];
        assert_eq!(contract.len(), 10);
        assert_eq!(
            map_security_status(-128),
            AuthorizationOutcome::AuthorizationCancelled
        );
        assert_eq!(
            map_security_status(-25293),
            AuthorizationOutcome::AuthorizationDenied
        );
        assert_eq!(
            map_security_status(-25308),
            AuthorizationOutcome::UserPresenceRequired
        );
        assert_eq!(
            map_security_status(-25300),
            AuthorizationOutcome::KeyMissing
        );
        assert_eq!(
            map_security_status(-34018),
            AuthorizationOutcome::UnsupportedAccessControl
        );
        assert_eq!(map_security_status(-1), AuthorizationOutcome::InternalError);
    }

    #[test]
    fn malformed_probe_refuses_before_keychain_access() {
        assert_eq!(
            probe_protected_creation("bad", &[0u8; 32]),
            ProbeOutcome::MalformedRequest
        );
        assert_eq!(
            probe_protected_creation(&"a".repeat(64), &[0u8; 7]),
            ProbeOutcome::MalformedRequest
        );
    }

    #[cfg(target_os = "macos")]
    #[test]
    #[ignore = "touches the real data-protection Keychain and requires a signed entitlement context"]
    fn actual_unsigned_context_refuses_protected_item() {
        let outcome = probe_protected_creation(&"f".repeat(64), &[0x5au8; 32]);
        assert_eq!(outcome, ProbeOutcome::UnsupportedAccessControl(-34018));
    }
}
