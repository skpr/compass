<?php

namespace App\Services;

use App\Models\Book;

class LibraryService
{
    private array $books = [];

    public function addBook(Book $book): void
    {
        $this->books[] = $book;
    }

    public function listBooks(): array
    {
        return $this->books;
    }
}