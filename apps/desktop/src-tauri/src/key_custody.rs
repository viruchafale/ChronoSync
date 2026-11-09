use ed25519_dalek::{Signer, SigningKey};
use serde::Serialize;
use sha2::{Digest, Sha256};
use std::fmt;
use std::sync::Mutex;
use zeroize::Zeroizing;

const SERVICE: &str = "local.chronosync.signing.v1";
const MAX_SIGNING_PAYLOAD: usize = 256 * 1024;

#[derive(Clone, Copy, Debug, PartialEq, Eq, Serialize)]
#[serde(rename_all = "snake_case")]
pub enum AccessStatus {
    Available,
    Denied,
    Missing,
    Corrupt,
    #[allow(dead_code)]
    Unsupported,
    RecoveryRequired,
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub enum CustodyError {
    Denied,
    Missing,
    Corrupt,
    #[allow(dead_code)]
    Unsupported,
    Conflict,
    InvalidRequest,
    Backend(i32),
}

pub struct CreatedKey {
    pub key_id: String,
    pub public_key: [u8; 32],
}

impl fmt::Display for CustodyError {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        let value = match self {
            Self::Denied => "KEY_ACCESS_DENIED",
            Self::Missing => "KEY_NOT_FOUND",
            Self::Corrupt => "KEY_CORRUPT",
            Self::Unsupported => "KEY_CUSTODY_UNSUPPORTED",
            Self::Conflict => "KEY_CUSTODY_CONFLICT",
            Self::InvalidRequest => "INVALID_SIGNING_REQUEST",
            Self::Backend(_) => "KEY_CUSTODY_BACKEND_ERROR",
        };
        formatter.write_str(value)
    }
}

pub trait KeyCustody: Send + Sync {
    fn create(&self) -> Result<CreatedKey, CustodyError>;
    fn import(&self, account: &str, seed: &[u8]) -> Result<[u8; 32], CustodyError>;
    fn public_key(&self, account: &str) -> Result<[u8; 32], CustodyError>;
    fn sign(&self, account: &str, payload: &[u8]) -> Result<[u8; 64], CustodyError>;
    fn status(&self, account: &str) -> AccessStatus;
    fn delete(&self, account: &str) -> Result<(), CustodyError>;
}

fn key_identifier(public_key: &[u8; 32]) -> String {
    let mut digest = Sha256::new();
    digest.update(b"chronosync-signing-key-v1\0ed25519\0");
    digest.update(public_key);
    digest
        .finalize()
        .iter()
        .map(|value| format!("{value:02x}"))
        .collect()
}

fn validate_account(account: &str) -> Result<(), CustodyError> {
    if account.len() != 64
        || !account
            .bytes()
            .all(|value| value.is_ascii_digit() || (b'a'..=b'f').contains(&value))
    {
        return Err(CustodyError::InvalidRequest);
    }
    Ok(())
}

fn signing_key(seed: &[u8]) -> Result<SigningKey, CustodyError> {
    SigningKey::try_from(seed).map_err(|_| CustodyError::Corrupt)
}

#[cfg(target_os = "macos")]
pub struct PlatformCustody {
    gate: Mutex<()>,
}

#[cfg(target_os = "macos")]
impl PlatformCustody {
    pub fn new() -> Self {
        Self {
            gate: Mutex::new(()),
        }
    }

    fn query(account: &str) -> security_framework::passwords::PasswordOptions {
        security_framework::passwords::PasswordOptions::new_generic_password(SERVICE, account)
    }

    fn read(account: &str) -> Result<Zeroizing<Vec<u8>>, CustodyError> {
        validate_account(account)?;
        security_framework::passwords::generic_password(Self::query(account))
            .map(Zeroizing::new)
            .map_err(map_keychain_error)
    }

    fn store_new(account: &str, seed: &[u8]) -> Result<(), CustodyError> {
        validate_account(account)?;
        if seed.len() != 32 {
            return Err(CustodyError::InvalidRequest);
        }
        match Self::read(account) {
            Ok(_) => return Err(CustodyError::Conflict),
            Err(CustodyError::Missing) => {}
            Err(error) => return Err(error),
        }
        security_framework::passwords::set_generic_password(SERVICE, account, seed)
            .map_err(map_keychain_error)
    }
}

#[cfg(target_os = "macos")]
fn map_keychain_error(error: security_framework::base::Error) -> CustodyError {
    match error.code() {
        -25300 => CustodyError::Missing,                // errSecItemNotFound
        -25293 | -25308 | -128 => CustodyError::Denied, // auth failed, interaction blocked, cancelled
        -25299 => CustodyError::Conflict,               // errSecDuplicateItem
        code => CustodyError::Backend(code),
    }
}

#[cfg(target_os = "macos")]
impl KeyCustody for PlatformCustody {
    fn create(&self) -> Result<CreatedKey, CustodyError> {
        let _guard = self.gate.lock().map_err(|_| CustodyError::Backend(0))?;
        let mut seed = Zeroizing::new([0u8; 32]);
        security_framework::random::SecRandom::default()
            .copy_bytes(&mut *seed)
            .map_err(|_| CustodyError::Backend(0))?;
        let key = SigningKey::from_bytes(&seed);
        let public = key.verifying_key().to_bytes();
        let key_id = key_identifier(&public);
        Self::store_new(&key_id, &seed[..])?;
        Ok(CreatedKey {
            key_id,
            public_key: public,
        })
    }

    fn import(&self, account: &str, seed: &[u8]) -> Result<[u8; 32], CustodyError> {
        let _guard = self.gate.lock().map_err(|_| CustodyError::Backend(0))?;
        validate_account(account)?;
        let key = signing_key(seed)?;
        let public = key.verifying_key().to_bytes();
        if key_identifier(&public) != account {
            return Err(CustodyError::Conflict);
        }
        match Self::read(account) {
            Ok(existing) => {
                let existing_key = signing_key(&existing)?;
                if existing_key.verifying_key().to_bytes() != public {
                    return Err(CustodyError::Conflict);
                }
                Ok(public)
            }
            Err(CustodyError::Missing) => {
                Self::store_new(account, seed)?;
                Ok(public)
            }
            Err(error) => Err(error),
        }
    }

    fn public_key(&self, account: &str) -> Result<[u8; 32], CustodyError> {
        let seed = Self::read(account)?;
        Ok(signing_key(&seed)?.verifying_key().to_bytes())
    }

    fn sign(&self, account: &str, payload: &[u8]) -> Result<[u8; 64], CustodyError> {
        if payload.is_empty() || payload.len() > MAX_SIGNING_PAYLOAD {
            return Err(CustodyError::InvalidRequest);
        }
        let seed = Self::read(account)?;
        Ok(signing_key(&seed)?.sign(payload).to_bytes())
    }

    fn status(&self, account: &str) -> AccessStatus {
        match self.public_key(account) {
            Ok(_) => AccessStatus::Available,
            Err(CustodyError::Denied) => AccessStatus::Denied,
            Err(CustodyError::Missing) => AccessStatus::Missing,
            Err(CustodyError::Unsupported) => AccessStatus::Unsupported,
            Err(CustodyError::Corrupt) => AccessStatus::Corrupt,
            Err(_) => AccessStatus::RecoveryRequired,
        }
    }

    fn delete(&self, account: &str) -> Result<(), CustodyError> {
        let _guard = self.gate.lock().map_err(|_| CustodyError::Backend(0))?;
        validate_account(account)?;
        security_framework::passwords::delete_generic_password(SERVICE, account)
            .map_err(map_keychain_error)
    }
}

#[cfg(not(target_os = "macos"))]
pub struct PlatformCustody;

#[cfg(not(target_os = "macos"))]
impl PlatformCustody {
    pub fn new() -> Self {
        Self
    }
}

#[cfg(not(target_os = "macos"))]
impl KeyCustody for PlatformCustody {
    fn create(&self) -> Result<CreatedKey, CustodyError> {
        Err(CustodyError::Unsupported)
    }
    fn import(&self, _: &str, _: &[u8]) -> Result<[u8; 32], CustodyError> {
        Err(CustodyError::Unsupported)
    }
    fn public_key(&self, _: &str) -> Result<[u8; 32], CustodyError> {
        Err(CustodyError::Unsupported)
    }
    fn sign(&self, _: &str, _: &[u8]) -> Result<[u8; 64], CustodyError> {
        Err(CustodyError::Unsupported)
    }
    fn status(&self, _: &str) -> AccessStatus {
        AccessStatus::Unsupported
    }
    fn delete(&self, _: &str) -> Result<(), CustodyError> {
        Err(CustodyError::Unsupported)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::collections::HashMap;
    use zeroize::Zeroize;

    #[derive(Default)]
    struct MemoryCustody {
        keys: Mutex<HashMap<String, Vec<u8>>>,
        denied: Mutex<bool>,
    }

    struct UnsupportedCustody;

    impl KeyCustody for UnsupportedCustody {
        fn create(&self) -> Result<CreatedKey, CustodyError> {
            Err(CustodyError::Unsupported)
        }
        fn import(&self, _: &str, _: &[u8]) -> Result<[u8; 32], CustodyError> {
            Err(CustodyError::Unsupported)
        }
        fn public_key(&self, _: &str) -> Result<[u8; 32], CustodyError> {
            Err(CustodyError::Unsupported)
        }
        fn sign(&self, _: &str, _: &[u8]) -> Result<[u8; 64], CustodyError> {
            Err(CustodyError::Unsupported)
        }
        fn status(&self, _: &str) -> AccessStatus {
            AccessStatus::Unsupported
        }
        fn delete(&self, _: &str) -> Result<(), CustodyError> {
            Err(CustodyError::Unsupported)
        }
    }

    impl KeyCustody for MemoryCustody {
        fn create(&self) -> Result<CreatedKey, CustodyError> {
            let public = signing_key(&[7u8; 32]).unwrap().verifying_key().to_bytes();
            let key_id = key_identifier(&public);
            self.import(&key_id, &[7u8; 32])?;
            Ok(CreatedKey {
                key_id,
                public_key: public,
            })
        }
        fn import(&self, account: &str, seed: &[u8]) -> Result<[u8; 32], CustodyError> {
            validate_account(account)?;
            let key = signing_key(seed)?;
            let public = key.verifying_key().to_bytes();
            if key_identifier(&public) != account {
                return Err(CustodyError::Conflict);
            }
            let mut keys = self.keys.lock().map_err(|_| CustodyError::Backend(0))?;
            if let Some(existing) = keys.get(account) {
                if signing_key(existing)?.verifying_key().to_bytes() != public {
                    return Err(CustodyError::Conflict);
                }
                return Ok(public);
            }
            keys.insert(account.into(), seed.to_vec());
            Ok(public)
        }
        fn public_key(&self, account: &str) -> Result<[u8; 32], CustodyError> {
            validate_account(account)?;
            if *self.denied.lock().map_err(|_| CustodyError::Backend(0))? {
                return Err(CustodyError::Denied);
            }
            let keys = self.keys.lock().map_err(|_| CustodyError::Backend(0))?;
            let seed = keys.get(account).ok_or(CustodyError::Missing)?;
            Ok(signing_key(seed)?.verifying_key().to_bytes())
        }
        fn sign(&self, account: &str, payload: &[u8]) -> Result<[u8; 64], CustodyError> {
            if payload.is_empty() || payload.len() > MAX_SIGNING_PAYLOAD {
                return Err(CustodyError::InvalidRequest);
            }
            if *self.denied.lock().map_err(|_| CustodyError::Backend(0))? {
                return Err(CustodyError::Denied);
            }
            let keys = self.keys.lock().map_err(|_| CustodyError::Backend(0))?;
            Ok(
                signing_key(keys.get(account).ok_or(CustodyError::Missing)?)?
                    .sign(payload)
                    .to_bytes(),
            )
        }
        fn status(&self, account: &str) -> AccessStatus {
            match self.public_key(account) {
                Ok(_) => AccessStatus::Available,
                Err(CustodyError::Denied) => AccessStatus::Denied,
                Err(CustodyError::Missing) => AccessStatus::Missing,
                _ => AccessStatus::Corrupt,
            }
        }
        fn delete(&self, account: &str) -> Result<(), CustodyError> {
            self.keys
                .lock()
                .map_err(|_| CustodyError::Backend(0))?
                .remove(account)
                .map(|mut seed| seed.zeroize())
                .ok_or(CustodyError::Missing)
        }
    }

    #[test]
    fn create_public_sign_delete_and_missing() {
        let custody = MemoryCustody::default();
        let created = custody.create().unwrap();
        let public = custody.public_key(&created.key_id).unwrap();
        assert_eq!(public, created.public_key);
        let signature = custody.sign(&created.key_id, b"bounded payload").unwrap();
        assert_eq!(signature.len(), 64);
        assert_eq!(custody.status(&created.key_id), AccessStatus::Available);
        custody.delete(&created.key_id).unwrap();
        assert_eq!(custody.status(&created.key_id), AccessStatus::Missing);
        assert_eq!(
            custody.sign(&created.key_id, b"payload"),
            Err(CustodyError::Missing)
        );
    }

    #[test]
    fn denial_conflict_idempotency_and_malformed_requests() {
        let custody = MemoryCustody::default();
        let account = key_identifier(&signing_key(&[1u8; 32]).unwrap().verifying_key().to_bytes());
        let public = custody.import(&account, &[1u8; 32]).unwrap();
        assert_eq!(custody.import(&account, &[1u8; 32]).unwrap(), public);
        assert_eq!(
            custody.import(&account, &[2u8; 32]),
            Err(CustodyError::Conflict)
        );
        *custody.denied.lock().unwrap() = true;
        assert_eq!(
            custody.sign(&account, b"payload"),
            Err(CustodyError::Denied)
        );
        assert_eq!(custody.status(&account), AccessStatus::Denied);
        assert_eq!(custody.public_key("bad"), Err(CustodyError::InvalidRequest));
        assert_eq!(
            custody.sign(&account, &[]),
            Err(CustodyError::InvalidRequest)
        );
    }

    #[test]
    fn unsupported_platform_is_explicit() {
        let custody = UnsupportedCustody;
        assert_eq!(custody.status("ignored"), AccessStatus::Unsupported);
        assert!(matches!(custody.create(), Err(CustodyError::Unsupported)));
        assert_eq!(
            custody.sign("ignored", b"payload"),
            Err(CustodyError::Unsupported)
        );
    }

    #[cfg(target_os = "macos")]
    #[test]
    #[ignore = "requires an interactive disposable macOS Keychain authorization ceremony"]
    fn real_keychain_round_trip() {
        let custody = PlatformCustody::new();
        let created = custody.create().unwrap();
        let public = custody.public_key(&created.key_id).unwrap();
        assert_eq!(public, created.public_key);
        assert_eq!(
            custody
                .sign(&created.key_id, b"ChronoSync Keychain integration proof")
                .unwrap()
                .len(),
            64
        );
        custody.delete(&created.key_id).unwrap();
    }
}
