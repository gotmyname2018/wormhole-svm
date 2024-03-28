//! Provide Types and Data about Wormhole's supported chains.

use std::{fmt, str::FromStr};

use serde::{Deserialize, Deserializer, Serialize, Serializer};
use thiserror::Error;

#[derive(Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub enum Chain {
    /// In the wormhole wire format, 0 indicates that a message is for any destination chain, it is
    /// represented here as `Any`.
    Any,

    /// Chains
    Solana,

    // Allow arbitrary u16s to support future chains.
    Unknown(u16),
}

impl From<u16> for Chain {
    fn from(other: u16) -> Chain {
        match other {
            0 => Chain::Any,
            1 => Chain::Solana,
            c => Chain::Unknown(c),
        }
    }
}

impl From<Chain> for u16 {
    fn from(other: Chain) -> u16 {
        match other {
            Chain::Any => 0,
            Chain::Solana => 1,
            Chain::Unknown(c) => c,
        }
    }
}

impl fmt::Display for Chain {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Any => f.write_str("Any"),
            Self::Solana => f.write_str("Solana"),
            Self::Unknown(v) => write!(f, "Unknown({v})"),
        }
    }
}

#[derive(Debug, Error)]
#[error("invalid chain: {0}")]
pub struct InvalidChainError(String);

impl FromStr for Chain {
    type Err = InvalidChainError;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s {
            "Any" | "any" | "ANY" => Ok(Chain::Any),
            "Solana" | "solana" | "SOLANA" => Ok(Chain::Solana),
            _ => {
                let mut parts = s.split(&['(', ')']);
                let _ = parts
                    .next()
                    .filter(|name| name.eq_ignore_ascii_case("unknown"))
                    .ok_or_else(|| InvalidChainError(s.into()))?;

                parts
                    .next()
                    .and_then(|v| v.parse::<u16>().ok())
                    .map(Chain::from)
                    .ok_or_else(|| InvalidChainError(s.into()))
            }
        }
    }
}

impl Default for Chain {
    fn default() -> Self {
        Self::Any
    }
}

impl Serialize for Chain {
    fn serialize<S>(&self, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: Serializer,
    {
        serializer.serialize_u16((*self).into())
    }
}

impl<'de> Deserialize<'de> for Chain {
    fn deserialize<D>(deserializer: D) -> Result<Self, D::Error>
    where
        D: Deserializer<'de>,
    {
        <u16 as Deserialize>::deserialize(deserializer).map(Self::from)
    }
}

#[cfg(test)]
mod test {
    use super::*;

    #[test]
    fn isomorphic_from() {
        for i in 0u16..=u16::MAX {
            assert_eq!(i, u16::from(Chain::from(i)));
        }
    }

    #[test]
    fn isomorphic_display() {
        for i in 0u16..=u16::MAX {
            let c = Chain::from(i);
            assert_eq!(c, c.to_string().parse().unwrap());
        }
    }
}
