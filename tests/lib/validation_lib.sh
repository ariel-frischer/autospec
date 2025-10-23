#!/usr/bin/env bash
# Validation Library for GitHub Issue Templates
# Version: 1.0.0
# Purpose: Validate YAML frontmatter and markdown structure

# validate_yaml_syntax: Validate YAML frontmatter syntax in template file
#
# Usage: validate_yaml_syntax <file_path>
# Returns: 0 if valid, 1 if invalid, 2 if validation tool unavailable
validate_yaml_syntax() {
  local file="$1"

  # Check file exists
  if [[ ! -f "$file" ]]; then
    echo "Error: File not found: $file" >&2
    return 1
  fi

  # Extract YAML frontmatter (between first two --- delimiters)
  local frontmatter
  frontmatter=$(sed -n '/^---$/,/^---$/p' "$file" | sed '1d;$d')

  if [[ -z "$frontmatter" ]]; then
    echo "Error: No YAML frontmatter found in $file" >&2
    return 1
  fi

  # Try yq first (supports both v3 and v4 syntax)
  if command -v yq >/dev/null 2>&1; then
    # Try yq v4 syntax first, then fallback to v3
    if echo "$frontmatter" | yq eval '.' - >/dev/null 2>&1; then
      return 0
    elif echo "$frontmatter" | yq '.' >/dev/null 2>&1; then
      return 0
    else
      echo "Error: Invalid YAML in $file" >&2
      return 1
    fi
  fi

  # Fallback to Python yaml module
  if command -v python3 >/dev/null 2>&1; then
    if echo "$frontmatter" | python3 -c "import sys, yaml; yaml.safe_load(sys.stdin)" >/dev/null 2>&1; then
      return 0
    else
      echo "Error: Invalid YAML in $file" >&2
      return 1
    fi
  fi

  echo "Error: No YAML validation tool available (yq or python3)" >&2
  return 2
}

# validate_required_fields: Verify required YAML frontmatter fields are present
#
# Usage: validate_required_fields <file_path>
# Returns: 0 if all required fields present, 1 if missing
validate_required_fields() {
  local file="$1"
  local missing_fields=()

  # Extract YAML frontmatter
  local frontmatter
  frontmatter=$(sed -n '/^---$/,/^---$/p' "$file" | sed '1d;$d')

  # Check 'name' field (support both yq v3 and v4)
  local name_value
  name_value=$(echo "$frontmatter" | yq eval '.name' - 2>/dev/null || echo "$frontmatter" | yq -r '.name' 2>/dev/null)
  if [[ -z "$name_value" || "$name_value" == "null" ]]; then
    missing_fields+=("name")
  fi

  # Check 'about' field (support both yq v3 and v4)
  local about_value
  about_value=$(echo "$frontmatter" | yq eval '.about' - 2>/dev/null || echo "$frontmatter" | yq -r '.about' 2>/dev/null)
  if [[ -z "$about_value" || "$about_value" == "null" ]]; then
    missing_fields+=("about")
  fi

  if [[ ${#missing_fields[@]} -gt 0 ]]; then
    echo "Error: Missing required fields in $file: ${missing_fields[*]}" >&2
    return 1
  fi

  return 0
}

# validate_template_sections: Verify required markdown sections are present
#
# Usage: validate_template_sections <file_path> <template_type>
# template_type: "bug_report" or "feature_request"
# Returns: 0 if all sections present, 1 if missing, 3 if invalid type
validate_template_sections() {
  local file="$1"
  local type="$2"
  local missing_sections=()

  case "$type" in
    bug_report)
      local required=(
        "## Bug Description"
        "## Steps to Reproduce"
        "## Expected Behavior"
        "## Actual Behavior"
        "## Environment"
      )
      ;;
    feature_request)
      local required=(
        "## Problem Statement"
        "## Use Case"
        "## Proposed Solution"
      )
      ;;
    *)
      echo "Error: Invalid template type: $type" >&2
      return 3
      ;;
  esac

  for section in "${required[@]}"; do
    if ! grep -qF "$section" "$file"; then
      missing_sections+=("$section")
    fi
  done

  if [[ ${#missing_sections[@]} -gt 0 ]]; then
    echo "Error: Missing sections in $file:" >&2
    printf '  - %s\n' "${missing_sections[@]}" >&2
    return 1
  fi

  return 0
}

# validate_config_file: Validate config.yml structure and syntax
#
# Usage: validate_config_file <file_path>
# Returns: 0 if valid, 1 if invalid, 2 if not found (optional file)
validate_config_file() {
  local file="$1"

  # Config file is optional
  if [[ ! -f "$file" ]]; then
    return 2
  fi

  # Validate YAML syntax for pure YAML file (not markdown)
  if command -v yq >/dev/null 2>&1; then
    # Try yq v4 syntax first, then v3
    if ! yq eval '.' "$file" >/dev/null 2>&1 && ! yq '.' "$file" >/dev/null 2>&1; then
      echo "Error: Invalid YAML in $file" >&2
      return 1
    fi
  elif command -v python3 >/dev/null 2>&1; then
    if ! python3 -c "import yaml; yaml.safe_load(open('$file'))" >/dev/null 2>&1; then
      echo "Error: Invalid YAML in $file" >&2
      return 1
    fi
  else
    echo "Error: No YAML validation tool available (yq or python3)" >&2
    return 2
  fi

  # Validate blank_issues_enabled if present (support both yq v3 and v4)
  local blank_issues
  blank_issues=$(yq -r '.blank_issues_enabled' "$file" 2>/dev/null || yq eval '.blank_issues_enabled' "$file" 2>/dev/null)
  if [[ "$blank_issues" != "null" && "$blank_issues" != "true" && "$blank_issues" != "false" ]]; then
    echo "Error: blank_issues_enabled must be boolean (true/false)" >&2
    return 1
  fi

  # Validate contact_links if present (support both yq v3 and v4)
  local links_count
  links_count=$(yq -r '.contact_links | length' "$file" 2>/dev/null || yq eval '.contact_links | length' "$file" 2>/dev/null)
  if [[ "$links_count" != "null" && "$links_count" -gt 0 ]]; then
    for ((i=0; i<links_count; i++)); do
      local name about url
      name=$(yq -r ".contact_links[$i].name" "$file" 2>/dev/null || yq eval ".contact_links[$i].name" "$file" 2>/dev/null)
      about=$(yq -r ".contact_links[$i].about" "$file" 2>/dev/null || yq eval ".contact_links[$i].about" "$file" 2>/dev/null)
      url=$(yq -r ".contact_links[$i].url" "$file" 2>/dev/null || yq eval ".contact_links[$i].url" "$file" 2>/dev/null)

      if [[ "$name" == "null" || "$about" == "null" || "$url" == "null" ]]; then
        echo "Error: contact_links[$i] missing required fields (name, url, about)" >&2
        return 1
      fi
    done
  fi

  return 0
}

# validate_all_templates: Run all validations on all template files
#
# Usage: validate_all_templates [template_dir]
# Returns: 0 if all valid, 1 if failures, 4 if directory not found
validate_all_templates() {
  local dir="${1:-.github/ISSUE_TEMPLATE}"
  local failed=0

  if [[ ! -d "$dir" ]]; then
    echo "Error: Template directory not found: $dir" >&2
    return 4
  fi

  # Validate config.yml
  echo "Validating config.yml..."
  local config_result
  validate_config_file "$dir/config.yml"
  config_result=$?
  case $config_result in
    0) echo "   Valid" ;;
    2) echo "  9 Not found (optional)" ;;
    *) echo "   Invalid"; ((failed++)) ;;
  esac

  # Validate bug_report.md
  if [[ -f "$dir/bug_report.md" ]]; then
    echo "Validating bug_report.md..."
    if validate_yaml_syntax "$dir/bug_report.md" && \
       validate_required_fields "$dir/bug_report.md" && \
       validate_template_sections "$dir/bug_report.md" bug_report; then
      echo "   Valid"
    else
      echo "   Invalid"
      ((failed++))
    fi
  fi

  # Validate feature_request.md
  if [[ -f "$dir/feature_request.md" ]]; then
    echo "Validating feature_request.md..."
    if validate_yaml_syntax "$dir/feature_request.md" && \
       validate_required_fields "$dir/feature_request.md" && \
       validate_template_sections "$dir/feature_request.md" feature_request; then
      echo "   Valid"
    else
      echo "   Invalid"
      ((failed++))
    fi
  fi

  if [[ $failed -eq 0 ]]; then
    echo " All templates valid"
    return 0
  else
    echo " $failed template(s) failed validation"
    return 1
  fi
}

# Functions are available when this file is sourced
