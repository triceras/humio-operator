#!/bin/bash

# Script to convert a local GitHub repository into a text document.
# The repository path is passed as a command-line argument.
# Each file's content is labeled with its file name.

# --- Configuration ---
# Set the output file name.
output_file="repo_content.txt"

# --- Script Start ---

# Check if a repository path was provided.
if [ $# -eq 0 ]; then
  echo "Error: No repository path provided."
  echo "Usage: $0 <repo_path>"
  exit 1
fi

# Get the repository path from the command-line argument.
repo_path="$1"

# Check if the repository path is valid.
if [ ! -d "$repo_path" ]; then
  echo "Error: Invalid repository path: $repo_path"
  exit 1
fi

# Change to the repository directory.
cd "$repo_path"

# Initialize the output file.
> "$output_file"

# Find all files in the repository (excluding .git directory).
find . -type f -not -path '*/.git/*' -print0 | while IFS= read -r -d $'\0' file; do
  # Extract the file name (relative path).
  file_name="$file"

  # Add the file name label to the output file.
  echo "file: $file_name" >> "$output_file"

  # Add the file content to the output file.
  cat "$file" >> "$output_file"

  # Add a separator between files (e.g., a newline).
  echo "" >> "$output_file"
done

echo "Repository content successfully written to: $output_file"

# --- Script End ---
