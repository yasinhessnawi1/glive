package entities

import (
	"github.com/glive/domain/values"
)

// Analysis represents the analysis result of a project
type Analysis struct {
	projectType        ProjectType
	detectedLanguages  []string
	packageManagers    []string
	entryPoints        []string
	dependencies       []Dependency
	systemRequirements []string
	commands           []*Command
	isSuspicious       bool
	suspiciousReasons  []string
	estimatedSize      string
	description        string
	usageInstructions  []string
	keyMilestones      []string
}

// Dependency represents a required dependency
type Dependency struct {
	Name      string
	Version   string
	Type      string // system, language, package
	Installed bool
}

// NewAnalysis creates a new Analysis entity
func NewAnalysis(projectType ProjectType) *Analysis {
	return &Analysis{
		projectType:        projectType,
		detectedLanguages:  []string{},
		packageManagers:    []string{},
		entryPoints:        []string{},
		dependencies:       []Dependency{},
		systemRequirements: []string{},
		commands:           []*Command{},
		isSuspicious:       false,
		suspiciousReasons:  []string{},
		estimatedSize:      "unknown",
		description:        "",
		usageInstructions:  []string{},
		keyMilestones:      []string{},
	}
}

// ProjectType returns the project type
func (a *Analysis) ProjectType() ProjectType {
	return a.projectType
}

// DetectedLanguages returns detected languages
func (a *Analysis) DetectedLanguages() []string {
	return a.detectedLanguages
}

// PackageManagers returns detected package managers
func (a *Analysis) PackageManagers() []string {
	return a.packageManagers
}

// EntryPoints returns detected entry points
func (a *Analysis) EntryPoints() []string {
	return a.entryPoints
}

// Dependencies returns detected dependencies
func (a *Analysis) Dependencies() []Dependency {
	return a.dependencies
}

// SystemRequirements returns system requirements
func (a *Analysis) SystemRequirements() []string {
	return a.systemRequirements
}

// Commands returns setup commands
func (a *Analysis) Commands() []*Command {
	return a.commands
}

// IsSuspicious returns whether the project is suspicious
func (a *Analysis) IsSuspicious() bool {
	return a.isSuspicious
}

// SuspiciousReasons returns reasons why the project is suspicious
func (a *Analysis) SuspiciousReasons() []string {
	return a.suspiciousReasons
}

// EstimatedSize returns the estimated project size
func (a *Analysis) EstimatedSize() string {
	return a.estimatedSize
}

// Description returns the project description
func (a *Analysis) Description() string {
	return a.description
}

// SetDetectedLanguages sets detected languages
func (a *Analysis) SetDetectedLanguages(languages []string) {
	a.detectedLanguages = languages
}

// SetPackageManagers sets package managers
func (a *Analysis) SetPackageManagers(managers []string) {
	a.packageManagers = managers
}

// SetEntryPoints sets entry points
func (a *Analysis) SetEntryPoints(entryPoints []string) {
	a.entryPoints = entryPoints
}

// SetDependencies sets dependencies
func (a *Analysis) SetDependencies(dependencies []Dependency) {
	a.dependencies = dependencies
}

// SetSystemRequirements sets system requirements
func (a *Analysis) SetSystemRequirements(requirements []string) {
	a.systemRequirements = requirements
}

// AddCommand adds a command to the analysis
func (a *Analysis) AddCommand(command *Command) {
	a.commands = append(a.commands, command)
}

// SetCommands sets all commands
func (a *Analysis) SetCommands(commands []*Command) {
	a.commands = commands
}

// SetSuspicious marks the analysis as suspicious with reasons
func (a *Analysis) SetSuspicious(reasons []string) {
	a.isSuspicious = true
	a.suspiciousReasons = reasons
}

// SetEstimatedSize sets the estimated size
func (a *Analysis) SetEstimatedSize(size string) {
	a.estimatedSize = size
}

// SetDescription sets the project description
func (a *Analysis) SetDescription(description string) {
	a.description = description
}

// UsageInstructions returns usage instructions
func (a *Analysis) UsageInstructions() []string {
	return a.usageInstructions
}

// SetUsageInstructions sets usage instructions
func (a *Analysis) SetUsageInstructions(instructions []string) {
	a.usageInstructions = instructions
}

// KeyMilestones returns key milestones
func (a *Analysis) KeyMilestones() []string {
	return a.keyMilestones
}

// SetKeyMilestones sets key milestones
func (a *Analysis) SetKeyMilestones(milestones []string) {
	a.keyMilestones = milestones
}

// HasCommands checks if the analysis has any commands
func (a *Analysis) HasCommands() bool {
	return len(a.commands) > 0
}

// RequiredCommands returns only required commands
func (a *Analysis) RequiredCommands() []*Command {
	var required []*Command
	for _, cmd := range a.commands {
		if cmd.Required() {
			required = append(required, cmd)
		}
	}
	return required
}

// CommandsByStage returns commands filtered by stage
func (a *Analysis) CommandsByStage(stage CommandStage) []*Command {
	var filtered []*Command
	for _, cmd := range a.commands {
		if cmd.Stage() == stage {
			filtered = append(filtered, cmd)
		}
	}
	return filtered
}

// Helper function to create a command from analysis data
// This will be used when converting from infrastructure analysis results
func NewCommandFromAnalysis(
	id, description, command string,
	workingDir *values.Path,
	stage CommandStage,
	required bool,
) (*Command, error) {
	return NewCommand(id, description, command, workingDir, stage, required)
}
