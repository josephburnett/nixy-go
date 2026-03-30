package file

import "testing"

func testFilesystem() *F {
	return &F{
		Type:             Folder,
		Owner:            OwnerRoot,
		OwnerPermission:  Write,
		CommonPermission: Read,
		Files: map[string]*F{
			"bin": {
				Type:             Folder,
				Owner:            OwnerRoot,
				OwnerPermission:  Write,
				CommonPermission: Read,
				Files: map[string]*F{
					"pwd": {
						Type:             Binary,
						Owner:            OwnerRoot,
						OwnerPermission:  Write,
						CommonPermission: Read,
						Data:             "pwd",
					},
				},
			},
			"home": {
				Type:             Folder,
				Owner:            OwnerRoot,
				OwnerPermission:  Write,
				CommonPermission: Read,
				Files: map[string]*F{
					"user": {
						Type:             Folder,
						Owner:            "user",
						OwnerPermission:  Write,
						CommonPermission: Read,
						Files:            map[string]*F{},
					},
				},
			},
			"etc": {
				Type:             Folder,
				Owner:            OwnerRoot,
				OwnerPermission:  Write,
				CommonPermission: None,
				Files: map[string]*F{
					"config": {
						Type:             Text,
						Owner:            OwnerRoot,
						OwnerPermission:  Write,
						CommonPermission: None,
						Data:             "key=value",
					},
				},
			},
		},
	}
}

func TestNavigate(t *testing.T) {
	fs := testFilesystem()

	// Navigate to root
	f, err := fs.Navigate([]string{})
	if err != nil {
		t.Fatal(err)
	}
	if f.Type != Folder {
		t.Fatalf("expected Folder, got %v", f.Type)
	}

	// Navigate to nested file
	f, err = fs.Navigate([]string{"bin", "pwd"})
	if err != nil {
		t.Fatal(err)
	}
	if f.Type != Binary {
		t.Fatalf("expected Binary, got %v", f.Type)
	}
	if f.Data != "pwd" {
		t.Fatalf("expected data 'pwd', got %v", f.Data)
	}

	// Navigate to nonexistent path
	_, err = fs.Navigate([]string{"bin", "nonexistent"})
	if err == nil {
		t.Fatal("expected error for nonexistent path")
	}

	// Navigate through non-folder
	_, err = fs.Navigate([]string{"bin", "pwd", "child"})
	if err == nil {
		t.Fatal("expected error navigating through non-folder")
	}
}

func TestCreateFile(t *testing.T) {
	fs := testFilesystem()

	// Create a file as owner
	newFile := &F{
		Type:             Text,
		Owner:            "user",
		OwnerPermission:  Write,
		CommonPermission: Read,
		Data:             "hello",
	}
	err := fs.CreateFile([]string{"home", "user"}, "hello.txt", newFile, "user")
	if err != nil {
		t.Fatal(err)
	}

	// Verify it exists
	f, err := fs.Navigate([]string{"home", "user", "hello.txt"})
	if err != nil {
		t.Fatal(err)
	}
	if f.Data != "hello" {
		t.Fatalf("expected 'hello', got %v", f.Data)
	}

	// Create duplicate fails
	err = fs.CreateFile([]string{"home", "user"}, "hello.txt", newFile, "user")
	if err == nil {
		t.Fatal("expected error for duplicate file")
	}

	// Create in non-folder fails
	err = fs.CreateFile([]string{"bin", "pwd"}, "file.txt", newFile, OwnerRoot)
	if err == nil {
		t.Fatal("expected error creating in non-folder")
	}
}

func TestCreateFilePermissions(t *testing.T) {
	fs := testFilesystem()

	newFile := &F{
		Type:            Text,
		Owner:           "user",
		OwnerPermission: Write,
	}

	// Non-owner cannot create in root-owned folder with no common write
	err := fs.CreateFile([]string{"etc"}, "new.txt", newFile, "user")
	if err == nil {
		t.Fatal("expected permission denied for non-owner")
	}

	// Root can create anywhere
	err = fs.CreateFile([]string{"etc"}, "new.txt", newFile, OwnerRoot)
	if err != nil {
		t.Fatal(err)
	}
}

func TestDeleteFile(t *testing.T) {
	fs := testFilesystem()

	// Delete as root
	err := fs.DeleteFile([]string{"bin"}, "pwd", OwnerRoot)
	if err != nil {
		t.Fatal(err)
	}

	// Verify gone
	_, err = fs.Navigate([]string{"bin", "pwd"})
	if err == nil {
		t.Fatal("expected error after deletion")
	}

	// Delete nonexistent
	err = fs.DeleteFile([]string{"bin"}, "nonexistent", OwnerRoot)
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestDeleteFilePermissions(t *testing.T) {
	fs := testFilesystem()

	// Non-owner cannot delete from root-owned /bin (common permission is Read)
	err := fs.DeleteFile([]string{"bin"}, "pwd", "user")
	if err == nil {
		t.Fatal("expected permission denied")
	}

	// Verify still exists
	_, err = fs.Navigate([]string{"bin", "pwd"})
	if err != nil {
		t.Fatal("file should still exist after denied delete")
	}
}

func TestSetData(t *testing.T) {
	fs := testFilesystem()

	// Root can set data on root-owned file
	err := fs.SetData([]string{"etc"}, "config", "newvalue", OwnerRoot)
	if err != nil {
		t.Fatal(err)
	}
	f, _ := fs.Navigate([]string{"etc", "config"})
	if f.Data != "newvalue" {
		t.Fatalf("expected 'newvalue', got %v", f.Data)
	}

	// Non-owner cannot write to root-owned file with no common write
	err = fs.SetData([]string{"etc"}, "config", "hacked", "user")
	if err == nil {
		t.Fatal("expected permission denied")
	}

	// SetData on nonexistent file
	err = fs.SetData([]string{"etc"}, "nonexistent", "data", OwnerRoot)
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestCanRead(t *testing.T) {
	f := &F{
		Owner:            OwnerRoot,
		OwnerPermission:  Write,
		CommonPermission: None,
	}
	if !f.CanRead(OwnerRoot) {
		t.Fatal("root should always be able to read")
	}
	if f.CanRead("user") {
		t.Fatal("non-owner should not read when common permission is None")
	}

	f.CommonPermission = Read
	if !f.CanRead("user") {
		t.Fatal("non-owner should read when common permission is Read")
	}
}

func TestCanWrite(t *testing.T) {
	f := &F{
		Owner:            "user",
		OwnerPermission:  Read,
		CommonPermission: None,
	}
	if f.CanWrite("user") {
		t.Fatal("owner with Read permission should not be able to write")
	}
	if !f.CanWrite(OwnerRoot) {
		t.Fatal("root should always be able to write")
	}

	f.OwnerPermission = Write
	if !f.CanWrite("user") {
		t.Fatal("owner with Write permission should be able to write")
	}
}
