<?php

// Manual class loading
require_once __DIR__ . '/../app/Models/Book.php';
require_once __DIR__ . '/../app/Services/LibraryService.php';

use App\Models\Book;
use App\Services\LibraryService;

// Create a new library service instance
$libraryService = new LibraryService();

// Add books to the library
$libraryService->addBook(new Book('The Catcher in the Rye', 'J.D. Salinger'));
$libraryService->addBook(new Book('To Kill a Mockingbird', 'Harper Lee'));

// List all books
echo "<h1>Library Books</h1>";
foreach ($libraryService->listBooks() as $book) {
    echo "<p>" . $book->getDetails() . "</p>";
}