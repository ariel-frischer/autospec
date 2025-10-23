#!/usr/bin/env bats
# Tests for config.yml

# Load validation library
source "$(dirname "$BATS_TEST_DIRNAME")/lib/validation_lib.sh"

CONFIG_PATH=".github/ISSUE_TEMPLATE/config.yml"

@test "config.yml file exists" {
  [ -f "$CONFIG_PATH" ]
}

@test "config.yml has valid YAML syntax" {
  # config.yml is pure YAML, not markdown with frontmatter
  if command -v yq >/dev/null 2>&1; then
    # Try yq v4 syntax first, then v3
    yq eval '.' "$CONFIG_PATH" >/dev/null 2>&1 || yq '.' "$CONFIG_PATH" >/dev/null 2>&1
  elif command -v python3 >/dev/null 2>&1; then
    python3 -c "import yaml; yaml.safe_load(open('$CONFIG_PATH'))" >/dev/null 2>&1
  else
    skip "No YAML validation tool available"
  fi
}

@test "config.yml has blank_issues_enabled field" {
  if command -v yq >/dev/null 2>&1; then
    # Check field exists (can be true or false)
    result=$(yq -r '.blank_issues_enabled' "$CONFIG_PATH" 2>/dev/null || yq eval '.blank_issues_enabled' "$CONFIG_PATH" 2>/dev/null)
    [ "$result" = "true" ] || [ "$result" = "false" ]
  else
    skip "yq not available"
  fi
}

@test "config.yml blank_issues_enabled is boolean" {
  run validate_config_file "$CONFIG_PATH"
  [ "$status" -eq 0 ]
}

@test "config.yml has contact_links array" {
  if command -v yq >/dev/null 2>&1; then
    # Check contact_links exists and is not null
    result=$(yq -r '.contact_links' "$CONFIG_PATH" 2>/dev/null || yq eval '.contact_links' "$CONFIG_PATH" 2>/dev/null)
    [ "$result" != "null" ]
  else
    skip "yq not available"
  fi
}

@test "config.yml contact_links have required fields (name, url, about)" {
  run validate_config_file "$CONFIG_PATH"
  [ "$status" -eq 0 ]
}

@test "config.yml has at least one contact link" {
  if command -v yq >/dev/null 2>&1; then
    count=$(yq -r '.contact_links | length' "$CONFIG_PATH" 2>/dev/null || yq eval '.contact_links | length' "$CONFIG_PATH" 2>/dev/null)
    [ "$count" -gt 0 ]
  else
    skip "yq not available"
  fi
}

@test "config.yml first contact link has name field" {
  if command -v yq >/dev/null 2>&1; then
    result=$(yq -r '.contact_links[0].name' "$CONFIG_PATH" 2>/dev/null || yq eval '.contact_links[0].name' "$CONFIG_PATH" 2>/dev/null)
    [ "$result" != "null" ]
    [ -n "$result" ]
  else
    skip "yq not available"
  fi
}

@test "config.yml first contact link has url field" {
  if command -v yq >/dev/null 2>&1; then
    result=$(yq -r '.contact_links[0].url' "$CONFIG_PATH" 2>/dev/null || yq eval '.contact_links[0].url' "$CONFIG_PATH" 2>/dev/null)
    [ "$result" != "null" ]
    [ -n "$result" ]
  else
    skip "yq not available"
  fi
}

@test "config.yml first contact link has about field" {
  if command -v yq >/dev/null 2>&1; then
    result=$(yq -r '.contact_links[0].about' "$CONFIG_PATH" 2>/dev/null || yq eval '.contact_links[0].about' "$CONFIG_PATH" 2>/dev/null)
    [ "$result" != "null" ]
    [ -n "$result" ]
  else
    skip "yq not available"
  fi
}
