# Contributing to nvx-go-driver

First off, thank you for considering contributing to **nvx-go-driver**! We welcome improvements, bug fixes, and new features. By participating in this project, you agree to abide by our guidelines.

## 🚀 Getting Started

We use a standard **Fork & Pull Request** workflow for contributions.

### 1. Fork the Repository
Click the **Fork** button at the top right corner of this repository's GitHub page to create a copy of the repository in your own GitHub account.

### 2. Clone Your Fork
Clone the forked repository to your local machine:
```bash
git clone https://github.com/<your-username>/nvx-go-driver.git
cd nvx-go-driver
```

### 3. Add Upstream Remote
To keep your fork in sync with the original repository, add the upstream remote. Assuming the original repo is owned by `Jkenyut` (adjust if different):
```bash
git remote add upstream https://github.com/Jkenyut/nvx-go-driver.git
```
*You can verify the remotes using `git remote -v`.*

### 4. Install Dependencies
```bash
make deps
```

---

## 🛠️ Development Workflow

### 1. Sync with Upstream
Before starting any work, ensure your local `main` branch is up to date with the upstream repository:
```bash
git checkout main
git fetch upstream
git rebase upstream/main
git push origin main
```

### 2. Create a Feature Branch
Create a new branch for your feature or bugfix. Use a descriptive name:
```bash
git checkout -b feature/your-feature-name
# or for a bug fix:
git checkout -b fix/your-bugfix-name
```

### 3. Make Your Changes
Write your code! Ensure your changes follow our coding standards.

### 4. Test and Lint
Before committing, make sure all tests pass and the code is properly formatted/linted:
```bash
# Run tests
make test

# Run linter
make lint
```

### 5. Commit Your Changes
We follow [Conventional Commits](https://www.conventionalcommits.org/). Please format your commit messages accordingly (e.g., `feat:`, `fix:`, `docs:`, `chore:`).
```bash
git add .
git commit -m "feat: add support for XYZ feature"
```

### 6. Push to Your Fork
```bash
git push -u origin feature/your-feature-name
```

### 7. Open a Pull Request (PR)
Go to the original `nvx-go-driver` repository on GitHub. You should see a prompt to open a Pull Request from your recently pushed branch. Provide a clear description of the changes in the PR.

---

## 📝 Coding Standards

- Follow standard idiomatic Go guidelines ([Effective Go](https://go.dev/doc/effective_go)).
- Ensure all public functions, structs, and interfaces have GoDocs.
- Add unit tests for any new functionality.
- Keep dependencies minimal and justified.
- Keep PRs focused on a single logical change.

## 🐛 Reporting Issues

If you find a bug or have a feature request, please search existing issues before opening a new one. Include:
- A clear, descriptive title.
- A detailed description of the issue.
- Steps to reproduce (if it's a bug).
- Expected vs. actual behavior.
- Environment details (Go version, OS, driver versions, etc.).
