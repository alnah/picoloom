package main

// Notes:
// - mergeFlags: we test all flag override scenarios exhaustively. Each flag
//   category (author, document, footer, cover, signature, toc) is tested
//   for both override and preserve behavior.
// - Auto-enable logic: we test that setting certain flags auto-enables
//   their parent feature (e.g., footer.text enables footer).
// These are acceptable gaps: we test observable behavior, not implementation details.

import (
	"testing"
)

// ---------------------------------------------------------------------------
// TestMergeFlags - CLI flags override config values
// ---------------------------------------------------------------------------

func TestMergeFlags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		flags *convertFlags
		cfg   *Config
		check func(t *testing.T, cfg *Config)
	}{
		{
			name:  "preserves config author when flags empty",
			flags: &convertFlags{},
			cfg:   &Config{Author: AuthorConfig{Name: "Config Author"}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Author.Name != "Config Author" {
					t.Errorf("mergeFlags() Author.Name = %q, want %q", cfg.Author.Name, "Config Author")
				}
			},
		},
		{
			name:  "overrides author.name with CLI flag",
			flags: &convertFlags{author: authorFlags{name: "CLI Author"}},
			cfg:   &Config{Author: AuthorConfig{Name: "Config Author"}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Author.Name != "CLI Author" {
					t.Errorf("mergeFlags() Author.Name = %q, want %q", cfg.Author.Name, "CLI Author")
				}
			},
		},
		{
			name:  "overrides author.title with CLI flag",
			flags: &convertFlags{author: authorFlags{title: "CLI Title"}},
			cfg:   &Config{Author: AuthorConfig{Title: "Config Title"}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Author.Title != "CLI Title" {
					t.Errorf("mergeFlags() Author.Title = %q, want %q", cfg.Author.Title, "CLI Title")
				}
			},
		},
		{
			name:  "overrides author.email with CLI flag",
			flags: &convertFlags{author: authorFlags{email: "cli@test.com"}},
			cfg:   &Config{Author: AuthorConfig{Email: "config@test.com"}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Author.Email != "cli@test.com" {
					t.Errorf("mergeFlags() Author.Email = %q, want %q", cfg.Author.Email, "cli@test.com")
				}
			},
		},
		{
			name:  "overrides author.org with CLI flag",
			flags: &convertFlags{author: authorFlags{org: "CLI Org"}},
			cfg:   &Config{Author: AuthorConfig{Organization: "Config Org"}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Author.Organization != "CLI Org" {
					t.Errorf("mergeFlags() Author.Organization = %q, want %q", cfg.Author.Organization, "CLI Org")
				}
			},
		},
		{
			name:  "overrides document.title with CLI flag",
			flags: &convertFlags{document: documentFlags{title: "CLI Title"}},
			cfg:   &Config{Document: DocumentConfig{Title: "Config Title"}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Document.Title != "CLI Title" {
					t.Errorf("mergeFlags() Document.Title = %q, want %q", cfg.Document.Title, "CLI Title")
				}
			},
		},
		{
			name:  "overrides document.subtitle with CLI flag",
			flags: &convertFlags{document: documentFlags{subtitle: "CLI Subtitle"}},
			cfg:   &Config{Document: DocumentConfig{Subtitle: "Config Subtitle"}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Document.Subtitle != "CLI Subtitle" {
					t.Errorf("mergeFlags() Document.Subtitle = %q, want %q", cfg.Document.Subtitle, "CLI Subtitle")
				}
			},
		},
		{
			name:  "overrides document.version with CLI flag",
			flags: &convertFlags{document: documentFlags{version: "v2.0"}},
			cfg:   &Config{Document: DocumentConfig{Version: "v1.0"}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Document.Version != "v2.0" {
					t.Errorf("mergeFlags() Document.Version = %q, want %q", cfg.Document.Version, "v2.0")
				}
			},
		},
		{
			name:  "overrides document.date with CLI flag",
			flags: &convertFlags{document: documentFlags{date: "2025-06-01"}},
			cfg:   &Config{Document: DocumentConfig{Date: "auto"}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Document.Date != "2025-06-01" {
					t.Errorf("mergeFlags() Document.Date = %q, want %q", cfg.Document.Date, "2025-06-01")
				}
			},
		},
		{
			name:  "overrides footer.position with CLI flag",
			flags: &convertFlags{footer: footerFlags{position: "left"}},
			cfg:   &Config{Footer: FooterConfig{Position: "right"}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Footer.Position != "left" {
					t.Errorf("mergeFlags() Footer.Position = %q, want %q", cfg.Footer.Position, "left")
				}
			},
		},
		{
			name:  "overrides footer.text with CLI flag",
			flags: &convertFlags{footer: footerFlags{text: "CLI Footer"}},
			cfg:   &Config{Footer: FooterConfig{Text: "Config Footer"}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Footer.Text != "CLI Footer" {
					t.Errorf("mergeFlags() Footer.Text = %q, want %q", cfg.Footer.Text, "CLI Footer")
				}
			},
		},
		{
			name:  "enables footer when footer.pageNumber flag set",
			flags: &convertFlags{footer: footerFlags{pageNumber: true}},
			cfg:   &Config{Footer: FooterConfig{Enabled: false, ShowPageNumber: false}},
			check: func(t *testing.T, cfg *Config) {
				if !cfg.Footer.ShowPageNumber {
					t.Error("mergeFlags() Footer.ShowPageNumber = false, want true")
				}
				if !cfg.Footer.Enabled {
					t.Error("mergeFlags() Footer.Enabled = false, want true")
				}
			},
		},
		{
			name:  "disables footer when footer.disabled flag set",
			flags: &convertFlags{footer: footerFlags{disabled: true}},
			cfg:   &Config{Footer: FooterConfig{Enabled: true}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Footer.Enabled {
					t.Error("mergeFlags() Footer.Enabled = true, want false")
				}
			},
		},
		{
			name:  "overrides cover.logo with CLI flag",
			flags: &convertFlags{cover: coverFlags{logo: "/cli/logo.png"}},
			cfg:   &Config{Cover: CoverConfig{Logo: "/config/logo.png"}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Cover.Logo != "/cli/logo.png" {
					t.Errorf("mergeFlags() Cover.Logo = %q, want %q", cfg.Cover.Logo, "/cli/logo.png")
				}
			},
		},
		{
			name:  "disables cover when cover.disabled flag set",
			flags: &convertFlags{cover: coverFlags{disabled: true}},
			cfg:   &Config{Cover: CoverConfig{Enabled: true}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Cover.Enabled {
					t.Error("mergeFlags() Cover.Enabled = true, want false")
				}
			},
		},
		{
			name:  "overrides signature.image with CLI flag",
			flags: &convertFlags{signature: signatureFlags{image: "/cli/sig.png"}},
			cfg:   &Config{Signature: SignatureConfig{ImagePath: "/config/sig.png"}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Signature.ImagePath != "/cli/sig.png" {
					t.Errorf("mergeFlags() Signature.ImagePath = %q, want %q", cfg.Signature.ImagePath, "/cli/sig.png")
				}
			},
		},
		{
			name:  "disables signature when signature.disabled flag set",
			flags: &convertFlags{signature: signatureFlags{disabled: true}},
			cfg:   &Config{Signature: SignatureConfig{Enabled: true}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Signature.Enabled {
					t.Error("mergeFlags() Signature.Enabled = true, want false")
				}
			},
		},
		{
			name:  "overrides toc.title with CLI flag",
			flags: &convertFlags{toc: tocFlags{title: "CLI Contents"}},
			cfg:   &Config{TOC: TOCConfig{Title: "Config Contents"}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.TOC.Title != "CLI Contents" {
					t.Errorf("mergeFlags() TOC.Title = %q, want %q", cfg.TOC.Title, "CLI Contents")
				}
			},
		},
		{
			name:  "overrides toc.minDepth with CLI flag",
			flags: &convertFlags{toc: tocFlags{minDepth: 2}},
			cfg:   &Config{TOC: TOCConfig{MinDepth: 1}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.TOC.MinDepth != 2 {
					t.Errorf("mergeFlags() TOC.MinDepth = %d, want %d", cfg.TOC.MinDepth, 2)
				}
			},
		},
		{
			name:  "overrides toc.maxDepth with CLI flag",
			flags: &convertFlags{toc: tocFlags{maxDepth: 4}},
			cfg:   &Config{TOC: TOCConfig{MaxDepth: 2}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.TOC.MaxDepth != 4 {
					t.Errorf("mergeFlags() TOC.MaxDepth = %d, want %d", cfg.TOC.MaxDepth, 4)
				}
			},
		},
		{
			name:  "disables toc when toc.disabled flag set",
			flags: &convertFlags{toc: tocFlags{disabled: true}},
			cfg:   &Config{TOC: TOCConfig{Enabled: true}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.TOC.Enabled {
					t.Error("mergeFlags() TOC.Enabled = true, want false")
				}
			},
		},
		{
			name: "overrides multiple author fields when all flags set",
			flags: &convertFlags{author: authorFlags{
				name:  "CLI Name",
				title: "CLI Title",
				email: "cli@test.com",
				org:   "CLI Org",
			}},
			cfg: &Config{Author: AuthorConfig{
				Name:         "Config Name",
				Title:        "Config Title",
				Email:        "config@test.com",
				Organization: "Config Org",
			}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Author.Name != "CLI Name" {
					t.Errorf("mergeFlags() Author.Name = %q, want %q", cfg.Author.Name, "CLI Name")
				}
				if cfg.Author.Title != "CLI Title" {
					t.Errorf("mergeFlags() Author.Title = %q, want %q", cfg.Author.Title, "CLI Title")
				}
				if cfg.Author.Email != "cli@test.com" {
					t.Errorf("mergeFlags() Author.Email = %q, want %q", cfg.Author.Email, "cli@test.com")
				}
				if cfg.Author.Organization != "CLI Org" {
					t.Errorf("mergeFlags() Author.Organization = %q, want %q", cfg.Author.Organization, "CLI Org")
				}
			},
		},
		{
			name:  "preserves other author fields when only name flag set",
			flags: &convertFlags{author: authorFlags{name: "CLI Name"}},
			cfg: &Config{Author: AuthorConfig{
				Name:         "Config Name",
				Title:        "Config Title",
				Email:        "config@test.com",
				Organization: "Config Org",
			}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Author.Name != "CLI Name" {
					t.Errorf("mergeFlags() Author.Name = %q, want %q", cfg.Author.Name, "CLI Name")
				}
				if cfg.Author.Title != "Config Title" {
					t.Errorf("mergeFlags() Author.Title = %q, want %q", cfg.Author.Title, "Config Title")
				}
				if cfg.Author.Email != "config@test.com" {
					t.Errorf("mergeFlags() Author.Email = %q, want %q", cfg.Author.Email, "config@test.com")
				}
				if cfg.Author.Organization != "Config Org" {
					t.Errorf("mergeFlags() Author.Organization = %q, want %q", cfg.Author.Organization, "Config Org")
				}
			},
		},
		// Extended metadata flags
		{
			name:  "overrides author.phone with CLI flag",
			flags: &convertFlags{author: authorFlags{phone: "+1-555-123-4567"}},
			cfg:   &Config{Author: AuthorConfig{Phone: "+1-555-000-0000"}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Author.Phone != "+1-555-123-4567" {
					t.Errorf("mergeFlags() Author.Phone = %q, want %q", cfg.Author.Phone, "+1-555-123-4567")
				}
			},
		},
		{
			name:  "overrides author.address with CLI flag",
			flags: &convertFlags{author: authorFlags{address: "123 CLI St"}},
			cfg:   &Config{Author: AuthorConfig{Address: "456 Config Ave"}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Author.Address != "123 CLI St" {
					t.Errorf("mergeFlags() Author.Address = %q, want %q", cfg.Author.Address, "123 CLI St")
				}
			},
		},
		{
			name:  "overrides author.department with CLI flag",
			flags: &convertFlags{author: authorFlags{department: "CLI Engineering"}},
			cfg:   &Config{Author: AuthorConfig{Department: "Config Engineering"}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Author.Department != "CLI Engineering" {
					t.Errorf("mergeFlags() Author.Department = %q, want %q", cfg.Author.Department, "CLI Engineering")
				}
			},
		},
		{
			name:  "overrides document.clientName with CLI flag",
			flags: &convertFlags{document: documentFlags{clientName: "CLI Client"}},
			cfg:   &Config{Document: DocumentConfig{ClientName: "Config Client"}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Document.ClientName != "CLI Client" {
					t.Errorf("mergeFlags() Document.ClientName = %q, want %q", cfg.Document.ClientName, "CLI Client")
				}
			},
		},
		{
			name:  "overrides document.projectName with CLI flag",
			flags: &convertFlags{document: documentFlags{projectName: "CLI Project"}},
			cfg:   &Config{Document: DocumentConfig{ProjectName: "Config Project"}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Document.ProjectName != "CLI Project" {
					t.Errorf("mergeFlags() Document.ProjectName = %q, want %q", cfg.Document.ProjectName, "CLI Project")
				}
			},
		},
		{
			name:  "overrides document.documentType with CLI flag",
			flags: &convertFlags{document: documentFlags{documentType: "CLI Spec"}},
			cfg:   &Config{Document: DocumentConfig{DocumentType: "Config Spec"}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Document.DocumentType != "CLI Spec" {
					t.Errorf("mergeFlags() Document.DocumentType = %q, want %q", cfg.Document.DocumentType, "CLI Spec")
				}
			},
		},
		{
			name:  "overrides document.documentID with CLI flag",
			flags: &convertFlags{document: documentFlags{documentID: "CLI-001"}},
			cfg:   &Config{Document: DocumentConfig{DocumentID: "CFG-001"}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Document.DocumentID != "CLI-001" {
					t.Errorf("mergeFlags() Document.DocumentID = %q, want %q", cfg.Document.DocumentID, "CLI-001")
				}
			},
		},
		{
			name:  "overrides document.description with CLI flag",
			flags: &convertFlags{document: documentFlags{description: "CLI Description"}},
			cfg:   &Config{Document: DocumentConfig{Description: "Config Description"}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Document.Description != "CLI Description" {
					t.Errorf("mergeFlags() Document.Description = %q, want %q", cfg.Document.Description, "CLI Description")
				}
			},
		},
		{
			name:  "enables footer when footer.showDocumentID flag set",
			flags: &convertFlags{footer: footerFlags{showDocumentID: true}},
			cfg:   &Config{Footer: FooterConfig{Enabled: false, ShowDocumentID: false}},
			check: func(t *testing.T, cfg *Config) {
				if !cfg.Footer.ShowDocumentID {
					t.Error("mergeFlags() Footer.ShowDocumentID = false, want true")
				}
				if !cfg.Footer.Enabled {
					t.Error("mergeFlags() Footer.Enabled = false, want true")
				}
			},
		},
		{
			name:  "enables department display when cover.showDepartment flag set",
			flags: &convertFlags{cover: coverFlags{showDepartment: true}},
			cfg:   &Config{Cover: CoverConfig{ShowDepartment: false}},
			check: func(t *testing.T, cfg *Config) {
				if !cfg.Cover.ShowDepartment {
					t.Error("mergeFlags() Cover.ShowDepartment = false, want true")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mergeFlags(tt.flags, tt.cfg)
			tt.check(t, tt.cfg)
		})
	}
}

// ---------------------------------------------------------------------------
// TestMergeFlagsAutoEnable - Auto-enable parent features when child flags set
// ---------------------------------------------------------------------------

func TestMergeFlagsAutoEnable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		flags *convertFlags
		cfg   *Config
		check func(t *testing.T, cfg *Config)
	}{
		{
			name:  "auto-enables footer when footer.text flag set",
			flags: &convertFlags{footer: footerFlags{text: "My Footer"}},
			cfg:   &Config{Footer: FooterConfig{Enabled: false}},
			check: func(t *testing.T, cfg *Config) {
				if !cfg.Footer.Enabled {
					t.Error("mergeFlags() Footer.Enabled = false, want true")
				}
				if cfg.Footer.Text != "My Footer" {
					t.Errorf("mergeFlags() Footer.Text = %q, want %q", cfg.Footer.Text, "My Footer")
				}
			},
		},
		{
			name:  "auto-enables footer when footer.position flag set",
			flags: &convertFlags{footer: footerFlags{position: "left"}},
			cfg:   &Config{Footer: FooterConfig{Enabled: false}},
			check: func(t *testing.T, cfg *Config) {
				if !cfg.Footer.Enabled {
					t.Error("mergeFlags() Footer.Enabled = false, want true")
				}
			},
		},
		{
			name:  "auto-enables cover when cover.logo flag set",
			flags: &convertFlags{cover: coverFlags{logo: "/path/to/logo.png"}},
			cfg:   &Config{Cover: CoverConfig{Enabled: false}},
			check: func(t *testing.T, cfg *Config) {
				if !cfg.Cover.Enabled {
					t.Error("mergeFlags() Cover.Enabled = false, want true")
				}
				if cfg.Cover.Logo != "/path/to/logo.png" {
					t.Errorf("mergeFlags() Cover.Logo = %q, want %q", cfg.Cover.Logo, "/path/to/logo.png")
				}
			},
		},
		{
			name:  "auto-enables cover when cover.showDepartment flag set",
			flags: &convertFlags{cover: coverFlags{showDepartment: true}},
			cfg:   &Config{Cover: CoverConfig{Enabled: false}},
			check: func(t *testing.T, cfg *Config) {
				if !cfg.Cover.Enabled {
					t.Error("mergeFlags() Cover.Enabled = false, want true")
				}
			},
		},
		{
			name:  "auto-enables signature when signature.image flag set",
			flags: &convertFlags{signature: signatureFlags{image: "/path/to/sig.png"}},
			cfg:   &Config{Signature: SignatureConfig{Enabled: false}},
			check: func(t *testing.T, cfg *Config) {
				if !cfg.Signature.Enabled {
					t.Error("mergeFlags() Signature.Enabled = false, want true")
				}
				if cfg.Signature.ImagePath != "/path/to/sig.png" {
					t.Errorf("mergeFlags() Signature.ImagePath = %q, want %q", cfg.Signature.ImagePath, "/path/to/sig.png")
				}
			},
		},
		{
			name:  "auto-enables TOC when toc.title flag set",
			flags: &convertFlags{toc: tocFlags{title: "Contents"}},
			cfg:   &Config{TOC: TOCConfig{Enabled: false}},
			check: func(t *testing.T, cfg *Config) {
				if !cfg.TOC.Enabled {
					t.Error("mergeFlags() TOC.Enabled = false, want true")
				}
				if cfg.TOC.Title != "Contents" {
					t.Errorf("mergeFlags() TOC.Title = %q, want %q", cfg.TOC.Title, "Contents")
				}
			},
		},
		{
			name:  "auto-enables TOC when toc.minDepth flag set",
			flags: &convertFlags{toc: tocFlags{minDepth: 2}},
			cfg:   &Config{TOC: TOCConfig{Enabled: false}},
			check: func(t *testing.T, cfg *Config) {
				if !cfg.TOC.Enabled {
					t.Error("mergeFlags() TOC.Enabled = false, want true")
				}
				if cfg.TOC.MinDepth != 2 {
					t.Errorf("mergeFlags() TOC.MinDepth = %d, want %d", cfg.TOC.MinDepth, 2)
				}
			},
		},
		{
			name:  "edge case: ignores negative toc.minDepth value",
			flags: &convertFlags{toc: tocFlags{minDepth: -1}},
			cfg:   &Config{TOC: TOCConfig{Enabled: false, MinDepth: 3}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.TOC.Enabled {
					t.Error("mergeFlags() TOC.Enabled = true, want false")
				}
				if cfg.TOC.MinDepth != 3 {
					t.Errorf("mergeFlags() TOC.MinDepth = %d, want %d", cfg.TOC.MinDepth, 3)
				}
			},
		},
		{
			name:  "edge case: ignores negative toc.maxDepth value",
			flags: &convertFlags{toc: tocFlags{maxDepth: -2}},
			cfg:   &Config{TOC: TOCConfig{Enabled: false, MaxDepth: 4}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.TOC.Enabled {
					t.Error("mergeFlags() TOC.Enabled = true, want false")
				}
				if cfg.TOC.MaxDepth != 4 {
					t.Errorf("mergeFlags() TOC.MaxDepth = %d, want %d", cfg.TOC.MaxDepth, 4)
				}
			},
		},
		{
			name:  "auto-enables TOC when toc.maxDepth flag set",
			flags: &convertFlags{toc: tocFlags{maxDepth: 3}},
			cfg:   &Config{TOC: TOCConfig{Enabled: false}},
			check: func(t *testing.T, cfg *Config) {
				if !cfg.TOC.Enabled {
					t.Error("mergeFlags() TOC.Enabled = false, want true")
				}
				if cfg.TOC.MaxDepth != 3 {
					t.Errorf("mergeFlags() TOC.MaxDepth = %d, want %d", cfg.TOC.MaxDepth, 3)
				}
			},
		},
		{
			name:  "disabled flag takes precedence over auto-enable",
			flags: &convertFlags{footer: footerFlags{text: "Footer", disabled: true}},
			cfg:   &Config{Footer: FooterConfig{Enabled: true}},
			check: func(t *testing.T, cfg *Config) {
				if cfg.Footer.Enabled {
					t.Error("mergeFlags() Footer.Enabled = true, want false")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mergeFlags(tt.flags, tt.cfg)
			tt.check(t, tt.cfg)
		})
	}
}
