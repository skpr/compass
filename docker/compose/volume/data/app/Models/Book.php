<?php

namespace App\Models;

class Book
{
    private string $title;
    private string $author;

    public function __construct(string $title, string $author)
    {
        $this->title = $title;
        $this->author = $author;
    }

    public function getTitle(): string
    {
        sleep(1);
        return $this->title;
    }

    public function getAuthor(): string
    {
        sleep(1);
        return $this->author;
    }

    public function getDetails(): string
    {
        sleep(2);
        return "{$this->title} by {$this->author}";
    }
}